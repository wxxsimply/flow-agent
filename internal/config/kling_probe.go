package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// KlingProbeResult 探测可灵 JWT 与 API 域名是否可用。
type KlingProbeResult struct {
	BaseURL string
	OK      bool
	Message string
}

var klingProbeOrigins = []string{
	"https://api-beijing.klingai.com",
	"https://api.klingai.com",
}

// 最小 1×1 PNG（用于探活，不实际生成视频）。
const probePNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

// ProbeKling 依次尝试国内/国际域名，POST 图生视频接口验证 AK/SK（参数错误亦视为认证通过）。
func ProbeKling(ctx context.Context, p Providers) []KlingProbeResult {
	ak := strings.TrimSpace(p.Kling.AccessKey)
	sk := strings.TrimSpace(p.Kling.SecretKey)
	if ak == "" || sk == "" {
		return []KlingProbeResult{{
			Message: "kling.access_key / secret_key 未配置",
		}}
	}
	origins := klingProbeOrigins
	custom := strings.TrimRight(strings.TrimSpace(p.Kling.BaseURL), "/")
	if custom != "" {
		origins = append([]string{custom}, origins...)
		seen := map[string]bool{}
		var uniq []string
		for _, o := range origins {
			if o == "" || seen[o] {
				continue
			}
			seen[o] = true
			uniq = append(uniq, o)
		}
		origins = uniq
	}
	var out []KlingProbeResult
	client := &http.Client{Timeout: 30 * time.Second}
	for _, base := range origins {
		r := probeKlingOrigin(ctx, client, base, ak, sk)
		out = append(out, r)
		if r.OK {
			break
		}
	}
	return out
}

func probeKlingOrigin(ctx context.Context, client *http.Client, base, ak, sk string) KlingProbeResult {
	r := KlingProbeResult{BaseURL: base}
	token, err := signKlingJWT(ak, sk)
	if err != nil {
		r.Message = fmt.Sprintf("JWT 签发失败: %v", err)
		return r
	}
	body, err := json.Marshal(map[string]any{
		"model_name": "kling-v2-1",
		"image":      probePNGBase64,
		"duration":   "5",
		"mode":       "std",
		"prompt":     "flowagent connectivity probe",
	})
	if err != nil {
		r.Message = err.Error()
		return r
	}
	url := base + "/v1/videos/image2video"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		r.Message = err.Error()
		return r
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		r.Message = err.Error()
		return r
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if isKlingAuthHTTP(resp.StatusCode, string(raw)) {
		r.Message = fmt.Sprintf("认证失败 HTTP %d: %s", resp.StatusCode, truncateKlingBody(string(raw)))
		return r
	}
	if resp.StatusCode == http.StatusOK {
		var envelope struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    struct {
				TaskID string `json:"task_id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Code == 0 && envelope.Data.TaskID != "" {
			r.OK = true
			r.Message = fmt.Sprintf("认证成功，探活任务已提交 task_id=%s", envelope.Data.TaskID)
			return r
		}
	}
	// 非 401：密钥有效，仅参数/余额等业务限制
	if resp.StatusCode != http.StatusUnauthorized {
		r.OK = true
		r.Message = fmt.Sprintf("认证通过（HTTP %d，可发起图生视频）", resp.StatusCode)
		if msg := parseKlingMsg(string(raw)); msg != "" {
			r.Message += " — " + msg
		}
		return r
	}
	r.Message = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncateKlingBody(string(raw)))
	return r
}

func isKlingAuthHTTP(status int, body string) bool {
	if status == http.StatusUnauthorized {
		return true
	}
	body = strings.ToLower(body)
	return strings.Contains(body, `"code":1000`) ||
		strings.Contains(body, `"code":1001`) ||
		strings.Contains(body, `"code":1002`) ||
		strings.Contains(body, "access key not found")
}

func parseKlingMsg(body string) string {
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(body), &envelope) == nil && envelope.Message != "" {
		return fmt.Sprintf("code=%d %s", envelope.Code, envelope.Message)
	}
	return ""
}

// signKlingJWT 与可灵官方示例一致：HS256，iss=AccessKey，nbf=now-5，exp=now+1800。
func signKlingJWT(accessKey, secretKey string) (string, error) {
	now := time.Now().Unix()
	claims := jwt.MapClaims{
		"iss": accessKey,
		"exp": now + 1800,
		"nbf": now - 5,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secretKey))
}

func truncateKlingBody(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

// FormatKlingProbeReport 生成 test-kling 人类可读报告。
func FormatKlingProbeReport(results []KlingProbeResult, ak string) string {
	var b strings.Builder
	mask := ak
	if len(mask) > 8 {
		mask = mask[:4] + "..." + mask[len(mask)-4:]
	}
	b.WriteString(fmt.Sprintf("kling access_key: %s\n\n", mask))
	for _, r := range results {
		mark := "[ ]"
		if r.OK {
			mark = "[x]"
		}
		b.WriteString(fmt.Sprintf("%s %s\n", mark, r.BaseURL))
		b.WriteString(fmt.Sprintf("    %s\n\n", r.Message))
	}
	for _, r := range results {
		if r.OK {
			b.WriteString(fmt.Sprintf("建议 base_url: %s\n", r.BaseURL))
			b.WriteString("重新编译后跑 produce：\n")
			b.WriteString("  go build -o bin\\flowagent.exe .\\cmd\\flowagent\n")
			b.WriteString("  .\\scripts\\accept-series-e2e.ps1 -LiveRun -AutoGate\n")
			return b.String()
		}
	}
	b.WriteString("全部域名认证失败。请确认：\n")
	b.WriteString("  1. 在 https://klingai.com/global/dev 创建 Access Key + Secret Key\n")
	b.WriteString("  2. access_key 与 secret_key 未填反\n")
	b.WriteString("  3. 密钥未过期、账户已开通 API 图生视频\n")
	return b.String()
}
