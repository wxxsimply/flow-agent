package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// 像素帧差启发式标定：YAVG 0–255，约 8–12 表示平滑微动。
const (
	heuristicDiffTarget    = 10.0
	heuristicMaxJumpWeight = 0.5
)

var (
	signalStatsYAVG     = regexp.MustCompile(`(?i)lavfi\.signalstats\.YAVG=([0-9.+-eE]+)`)
	signalStatsYAVGAlt  = regexp.MustCompile(`(?i)YAVG[=:]([0-9.+-eE]+)`)
	psnrMSEPattern      = regexp.MustCompile(`(?i)MSE[=:]([0-9.+-eE]+)`)
)

var (
	bonScoreMu sync.Mutex
	bonScores  []artifacts.BoNScoreEntry
)

func resetBonScores() {
	bonScoreMu.Lock()
	bonScores = nil
	bonScoreMu.Unlock()
}

func recordBonScore(e artifacts.BoNScoreEntry) {
	bonScoreMu.Lock()
	bonScores = append(bonScores, e)
	bonScoreMu.Unlock()
}

func bonScoresLocked() []artifacts.BoNScoreEntry {
	out := make([]artifacts.BoNScoreEntry, len(bonScores))
	copy(out, bonScores)
	return out
}

type physicsScoreResult struct {
	Score      float64
	Unreliable bool
	Scorer     string
}

// imageToVideoFileBoN 生成多条 i2v 候选，按物理可信度选优（WMReward 脚本或启发式）。
func imageToVideoFileBoN(
	ctx context.Context,
	rc *runctx.Context,
	vidCfg config.StackVideoConfig,
	imgPath, prompt string,
	durSec float64,
	outPath string,
	shot *artifacts.Shot,
) error {
	if rc.DryRun || !vidCfg.WMRewardBoNEnabled || vidCfg.WMRewardBoNCandidates < 2 {
		return imageToVideoFile(ctx, rc, vidCfg, imgPath, prompt, durSec, outPath)
	}
	if checkAccountOverdue(rc) {
		return ErrMediaAccountOverdue
	}

	n := vidCfg.WMRewardBoNCandidates
	if n > 5 {
		n = 5
	}
	dir := filepath.Join(filepath.Dir(outPath), "_bon_"+strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath)))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	bestPath := ""
	bestScore := math.MaxFloat64
	bestIdx := 0
	var candScores []artifacts.BoNCandidateScore
	scorer := bonScorerName(vidCfg)
	allUnreliable := true
	scoresDiscriminate := false

	for i := 0; i < n; i++ {
		if i > 0 && allUnreliable {
			slog.Info("wmreward bon skip remaining candidates", "reason", "unreliable_scores")
			break
		}
		cand := filepath.Join(dir, fmt.Sprintf("cand-%02d.mp4", i+1))
		p := promptVariantForBoN(prompt, shot, i)
		if err := imageToVideoFile(ctx, rc, vidCfg, imgPath, p, durSec, cand); err != nil {
			if errors.Is(err, ErrUseKenBurns) {
				return err
			}
			if errors.Is(err, ErrMediaAccountOverdue) {
				return err
			}
			slog.Warn("wmreward bon candidate failed", "i", i+1, "err", err)
			continue
		}
		ps, err := scoreClipPhysicsDetailed(vidCfg, cand)
		score := float64(i)
		unreliable := true
		if err == nil {
			score = ps.Score
			unreliable = ps.Unreliable
			if ps.Scorer != "" {
				scorer = ps.Scorer
			}
		} else {
			slog.Warn("wmreward score failed", "cand", cand, "err", err)
		}
		if !unreliable {
			allUnreliable = false
		}
		slog.Info("wmreward bon candidate", "i", i+1, "score", score, "lower_is_better", true, "scorer", scorer, "unreliable", unreliable)
		candScores = append(candScores, artifacts.BoNCandidateScore{
			Index:      i + 1,
			Score:      score,
			Path:       cand,
			Unreliable: unreliable,
		})
		if bestPath == "" || score < bestScore {
			if bestPath != "" && !unreliable && !candScores[len(candScores)-2].Unreliable && score != bestScore {
				scoresDiscriminate = true
			}
			bestPath = cand
			bestScore = score
			bestIdx = i + 1
		} else if !unreliable && score != bestScore {
			scoresDiscriminate = true
		}
		if i == 0 && allUnreliable {
			break
		}
		if i >= 1 && !scoresDiscriminate && !allUnreliable {
			// 前两候选分数相同且可靠 → 不再生成额外候选
			break
		}
	}
	if bestPath == "" {
		return imageToVideoFile(ctx, rc, vidCfg, imgPath, prompt, durSec, outPath)
	}

	entry := artifacts.BoNScoreEntry{
		ClipPath:           outPath,
		Scorer:             scorer,
		Candidates:         candScores,
		Selected:           bestIdx,
		SelectedScore:      bestScore,
		SelectedUnreliable: allUnreliable,
	}
	if shot != nil {
		entry.ShotID = shot.ID
	}
	recordBonScore(entry)

	scoresJSON, _ := json.MarshalIndent(entry, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "scores.json"), scoresJSON, 0o644)

	data, err := os.ReadFile(bestPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		return err
	}
	sel := fmt.Sprintf("%s\nscore=%.6f\nscorer=%s\nunreliable=%v\n", bestPath, bestScore, scorer, allUnreliable)
	_ = os.WriteFile(filepath.Join(dir, "selected.txt"), []byte(sel), 0o644)
	return nil
}

func bonScorerName(vidCfg config.StackVideoConfig) string {
	if script := strings.TrimSpace(vidCfg.WMRewardScriptPath); script != "" {
		if _, err := os.Stat(script); err == nil {
			return "wmreward_script"
		}
	}
	if env := strings.TrimSpace(os.Getenv("FLOWAGENT_WMREWARD_SCRIPT")); env != "" {
		if _, err := os.Stat(env); err == nil {
			return "wmreward_script"
		}
	}
	return "pixel_heuristic"
}

func promptVariantForBoN(base string, shot *artifacts.Shot, idx int) string {
	out := base
	if shot == nil {
		return out + bonVariantSuffix(idx, "", "")
	}
	cues := strings.TrimSpace(shot.PhysicsCues)
	forbidden := firstForbiddenItems(shot.ForbiddenPhysics, 2)
	return out + bonVariantSuffix(idx, cues, forbidden)
}

func bonVariantSuffix(idx int, cues, forbidden string) string {
	switch idx {
	case 0:
		s := "，固定机位无推拉摇移"
		if cues != "" {
			s += "，" + truncateRunesStr(cues, 80)
		}
		return s
	case 1:
		s := "，仅单一主动作、幅度极小，预备与收势静止"
		if cues != "" {
			s += "，" + truncateRunesStr(cues, 60)
		}
		return s
	case 2:
		s := "，固定机位，动作连贯符合重力与支撑"
		if forbidden != "" {
			s += "，禁止：" + forbidden
		}
		return s
	case 3:
		s := "，全程道具固定在同一手，形状刚性不变"
		if forbidden != "" {
			s += "，禁止：" + forbidden
		}
		if cues != "" {
			s += "，" + truncateRunesStr(cues, 40)
		}
		return s
	default:
		s := "，因果顺序正确：先接触后形变，禁止未触即动"
		if cues != "" {
			s += "，" + truncateRunesStr(cues, 40)
		}
		return s
	}
}

func firstForbiddenItems(forbidden string, n int) string {
	if n <= 0 {
		return ""
	}
	parts := strings.FieldsFunc(forbidden, func(r rune) bool {
		return r == '，' || r == ',' || r == '；' || r == ';' || r == '、'
	})
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
		if len(out) >= n {
			break
		}
	}
	return strings.Join(out, "、")
}

func truncateRunesStr(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
		return string(r)
	}
	return string(r[:max])
}

func scoreClipPhysicsDetailed(vidCfg config.StackVideoConfig, videoPath string) (physicsScoreResult, error) {
	scorer := bonScorerName(vidCfg)
	if script := strings.TrimSpace(vidCfg.WMRewardScriptPath); script != "" {
		if s, err := scoreViaWMRewardScript(script, videoPath); err == nil {
			return physicsScoreResult{Score: s, Scorer: "wmreward_script"}, nil
		}
	}
	if env := strings.TrimSpace(os.Getenv("FLOWAGENT_WMREWARD_SCRIPT")); env != "" {
		if s, err := scoreViaWMRewardScript(env, videoPath); err == nil {
			return physicsScoreResult{Score: s, Scorer: "wmreward_script"}, nil
		}
	}
	s, err := scoreClipHeuristic(videoPath)
	if err != nil {
		return physicsScoreResult{}, err
	}
	return physicsScoreResult{Score: s, Scorer: scorer}, nil
}

// scoreClipPhysics 越低越优。优先外部 WMReward 脚本，否则帧差启发式。
func scoreClipPhysics(rc *runctx.Context, vidCfg config.StackVideoConfig, videoPath string) (float64, error) {
	_ = rc
	ps, err := scoreClipPhysicsDetailed(vidCfg, videoPath)
	if err != nil {
		return 0, err
	}
	return ps.Score, nil
}

func scoreViaWMRewardScript(scriptPath, videoPath string) (float64, error) {
	scriptPath, err := validateWMRewardScriptPath(scriptPath)
	if err != nil {
		return 0, err
	}
	cmd := exec.Command("python", scriptPath, videoPath)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var surprise float64
	if _, scanErr := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &surprise); scanErr != nil {
		return 0, scanErr
	}
	return surprise, nil
}

// scoreClipHeuristic 用帧间像素差估计：过高=闪烁/剧烈运动，过低=静止；取接近目标的候选。
func scoreClipHeuristic(videoPath string) (float64, error) {
	tmp, err := os.MkdirTemp("", "flowagent-frames-*")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)

	pattern := filepath.Join(tmp, "f%03d.png")
	// fps=2 保证短 clip 也能提取足够帧（Windows 兼容）
	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-vf", "fps=2", "-frames:v", "8", pattern)
	if out, err := cmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("ffmpeg frames: %w: %s", err, string(out))
	}
	var diffs []float64
	var prev string
	for i := 1; i <= 8; i++ {
		fp := filepath.Join(tmp, fmt.Sprintf("f%03d.png", i))
		if _, err := os.Stat(fp); err != nil {
			continue
		}
		if prev != "" {
			d, err := frameMeanAbsDiff(prev, fp)
			if err == nil {
				diffs = append(diffs, d)
			}
		}
		prev = fp
	}
	if len(diffs) == 0 {
		return 0, fmt.Errorf("heuristic: no frame diffs computed")
	}
	var sum float64
	for _, d := range diffs {
		sum += d
	}
	mean := sum / float64(len(diffs))
	score := abs(mean - heuristicDiffTarget)
	var maxJump float64
	for i := 1; i < len(diffs); i++ {
		if j := abs(diffs[i] - diffs[i-1]); j > maxJump {
			maxJump = j
		}
	}
	return score + maxJump*heuristicMaxJumpWeight, nil
}

func frameMeanAbsDiff(pathA, pathB string) (float64, error) {
	cmd := exec.Command("ffmpeg", "-y", "-hide_banner", "-loglevel", "info", "-i", pathA, "-i", pathB,
		"-lavfi", "[0:v][1:v]blend=all_mode=difference,format=gray,signalstats",
		"-frames:v", "1", "-f", "null", "-")
	out, err := cmd.CombinedOutput()
	combined := string(out)
	if err != nil {
		// signalstats 可能仍输出 YAVG 到 stderr
		if yavg, ok := parseFrameDiffMetric(combined); ok {
			return yavg, nil
		}
		return 0, fmt.Errorf("ffmpeg diff: %w: %s", err, combined)
	}
	yavg, ok := parseFrameDiffMetric(combined)
	if !ok {
		return 0, fmt.Errorf("ffmpeg diff: metric not found in output")
	}
	return yavg, nil
}

func parseFrameDiffMetric(output string) (float64, bool) {
	if v, ok := parseSignalStatsYAVG(output); ok {
		return v, true
	}
	if m := psnrMSEPattern.FindStringSubmatch(output); len(m) >= 2 {
		var mse float64
		if _, err := fmt.Sscanf(m[1], "%f", &mse); err == nil && mse >= 0 {
			// MSE 越大差异越大；缩放到与 YAVG 相近量级
			return math.Sqrt(mse), true
		}
	}
	return 0, false
}

func parseSignalStatsYAVG(output string) (float64, bool) {
	for _, re := range []*regexp.Regexp{signalStatsYAVG, signalStatsYAVGAlt} {
		m := re.FindStringSubmatch(output)
		if len(m) >= 2 {
			var v float64
			if _, err := fmt.Sscanf(m[1], "%f", &v); err == nil {
				return v, true
			}
		}
	}
	return 0, false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func validateWMRewardScriptPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty wmreward script path")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", fmt.Errorf("wmreward script path is a directory")
	}
	if !strings.HasSuffix(strings.ToLower(absPath), ".py") {
		return "", fmt.Errorf("wmreward script must be a .py file")
	}
	return absPath, nil
}
