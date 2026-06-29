package agent

import (
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"

	ve "github.com/flow-agent/flow-agent/internal/provider/volcengine"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

// ErrUseKenBurns 表示跳过 AI i2v，改用关键帧 Ken Burns 生成 mp4。
var ErrUseKenBurns = errors.New("use ken burns clip")

type produceState struct {
	mu              sync.Mutex
	accountOverdue  bool
}

var produceStates sync.Map // runID -> *produceState

func initProduceState(rc *runctx.Context) {
	if rc == nil || rc.RunID == "" {
		return
	}
	st := &produceState{}
	if skipI2VFromEnv() {
		slog.Info("produce: AI i2v disabled, using Ken Burns for motion clips", "reason", "FLOWAGENT_SKIP_I2V")
	}
	produceStates.Store(rc.RunID, st)
}

func markAccountOverdue(rc *runctx.Context) {
	st := getProduceState(rc)
	if st == nil {
		return
	}
	st.mu.Lock()
	st.accountOverdue = true
	st.mu.Unlock()
}

func checkAccountOverdue(rc *runctx.Context) bool {
	st := getProduceState(rc)
	if st == nil {
		return false
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.accountOverdue
}

func skipI2VFromEnv() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("FLOWAGENT_SKIP_I2V")))
	return v == "1" || v == "true" || v == "yes"
}

func clearProduceState(rc *runctx.Context) {
	if rc == nil || rc.RunID == "" {
		return
	}
	produceStates.Delete(rc.RunID)
}

func getProduceState(rc *runctx.Context) *produceState {
	if rc == nil || rc.RunID == "" {
		return nil
	}
	if v, ok := produceStates.Load(rc.RunID); ok {
		if st, ok := v.(*produceState); ok {
			return st
		}
	}
	return nil
}

// shouldSkipI2V 仅环境变量 FLOWAGENT_SKIP_I2V 会全局跳过；单镜 API 失败不再污染整 run。
func shouldSkipI2V(rc *runctx.Context) bool {
	return skipI2VFromEnv()
}

// markSkipI2V 保留日志接口；不再设置 run 级 skip 开关。
func markSkipI2V(rc *runctx.Context, reason string) {
	if strings.TrimSpace(reason) == "" {
		return
	}
	slog.Debug("produce: i2v unavailable for shot", "reason", strings.TrimSpace(reason))
}

func isWanFatal(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "accessdenied") ||
		strings.Contains(msg, "access denied") ||
		strings.Contains(msg, "invalid_api_key") ||
		strings.Contains(msg, "invalid api key") ||
		strings.Contains(msg, "permission denied")
}

func isVolcengineFatalForShot(err error) bool {
	return err != nil && ve.IsVolcengineFatal(err)
}

func imageToVideoShouldKenBurns(rc *runctx.Context, err error) bool {
	if err == nil {
		return false
	}
	if shouldSkipI2V(rc) {
		return true
	}
	// 账户欠费：require_video 栈必须失败，禁止静默 Ken Burns 跑完全片
	if isArrearage(err) {
		if rc != nil && rc.App != nil && rc.App.Stack != nil && rc.App.Stack.VideoConfig().RequireVideo {
			return false
		}
		return true
	}
	if isVolcengineFatalForShot(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "modelnotopen")
}

func noteProduceAPIError(rc *runctx.Context, shotID string, err error) {
	if rc == nil || err == nil {
		return
	}
	if errors.Is(err, ErrMediaAccountOverdue) || isArrearage(err) {
		markAccountOverdue(rc)
	}
	slog.Warn("produce api error", "shot", shotID, "err", err)
}
