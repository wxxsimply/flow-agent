package runner

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/agent"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func directorMode(rc *runctx.Context) bool {
	if rc.Workflow != "micro-movie" {
		return false
	}
	agent.LoadCreativeOptionsFromRun(rc)
	return rc.Creative != nil && rc.Creative.IsDirector()
}

// CheckGates 阶段结束后校验 YAML 中声明的门禁。
func CheckGates(rc *runctx.Context, stage *workflow.StageDefinition) error {
	for _, g := range stage.Gates {
		switch g.Type {
		case "human":
			if rc.AutoGate {
				rc.SetGate(g.ID, true) // 开发模式自动放行
				continue
			}
			if deferHumanGateForReview(rc, stage, g) {
				continue
			}
			if !rc.GatePassed(g.ID) {
				return fmt.Errorf("human gate %q not approved (use --auto-gate for dev)", g.ID)
			}
		case "automatic":
			var err error
			if strings.TrimSpace(g.Condition) != "" {
				err = EvaluateGateCondition(rc, g)
			} else {
				err = checkAutomaticGate(rc, g.ID)
			}
			if err != nil {
				return err
			}
			rc.SetGate(g.ID, true)
		default:
			if !rc.GatePassed(g.ID) && g.Type != "" {
				return fmt.Errorf("gate %q not satisfied", g.ID)
			}
		}
	}
	return nil
}

// checkAutomaticGate 按门禁 id 做简单自动校验（后续可接表达式引擎）。
func checkAutomaticGate(rc *runctx.Context, id string) error {
	switch id {
	case "continuity_passed":
		if rc.DryRun {
			break
		}
		report, err := artifacts.LoadContinuityReport(rc.ArtifactPath("artifacts/continuity-report.json"))
		if err != nil {
			return fmt.Errorf("gate %s: %w", id, err)
		}
		if report.CriticalCount > 0 {
			return fmt.Errorf("gate %s: %d critical issue(s)", id, report.CriticalCount)
		}
	case "duration_ok":
		if err := checkDurationOK(rc); err != nil {
			return err
		}
	case "no_block_issues":
		if err := checkNoBlockIssues(rc); err != nil {
			return err
		}
	case "length_in_range":
		if rc.DryRun {
			break
		}
		if err := checkLengthInRange(rc); err != nil {
			return err
		}
	case "av_sync_ok":
		if err := checkAVSyncOK(rc); err != nil {
			return err
		}
	case "motion_quality_ok":
		if err := checkMotionQualityOK(rc); err != nil {
			return err
		}
	case "audio_duration_ok":
		if err := checkAudioDurationOK(rc); err != nil {
			return err
		}
	case "narration_complete_ok":
		if err := checkNarrationCompleteOK(rc); err != nil {
			return err
		}
	case "visual_quality_ok":
		if err := checkVisualQualityOK(rc); err != nil {
			return err
		}
	}
	return nil
}

func checkAVSyncOK(rc *runctx.Context) error {
	const gateID = "av_sync_ok"
	path := rc.ArtifactPath("artifacts/sync-report.json")
	report, err := artifacts.LoadSyncReport(path)
	if err != nil {
		return fmt.Errorf("gate %s: %w", gateID, err)
	}
	if report.MaxDriftSec > 0.5 {
		return fmt.Errorf("gate %s: max_drift_sec=%.3f (want <=0.5)", gateID, report.MaxDriftSec)
	}
	return nil
}

// checkDurationOK 校验 storyboard 总时长与目标误差 ≤3s。
func checkDurationOK(rc *runctx.Context) error {
	path := rc.ArtifactPath("artifacts/storyboard.json")
	sb, err := artifacts.LoadStoryboard(path)
	if err != nil {
		return fmt.Errorf("gate duration_ok: %w", err)
	}
	if directorMode(rc) {
		if len(sb.Shots) == 0 {
			return fmt.Errorf("gate duration_ok: no shots in director mode")
		}
		if sb.TotalDurationSec() <= 0 {
			return fmt.Errorf("gate duration_ok: invalid total duration")
		}
		// director：时长以 TTS 估计为准，target 仅作参考；produce 阶段用 audio_duration_ok
		return nil
	}
	target := float64(rc.TargetDurationSec())
	total := sb.TotalDurationSec()
	tol := artifacts.DurationToleranceSec
	if rc.Workflow == "micro-movie" {
		tol = 15
	}
	if math.Abs(total-target) > tol {
		return fmt.Errorf("gate duration_ok: total %.1fs, want %.1f±%.0fs", total, target, tol)
	}
	return nil
}

// checkNoBlockIssues 校验 compliance-report.json 无 block 级违规。
func checkNoBlockIssues(rc *runctx.Context) error {
	const gateID = "no_block_issues"
	path := rc.ArtifactPath("artifacts/compliance-report.json")
	report, err := artifacts.LoadComplianceReport(path)
	if err != nil {
		return fmt.Errorf("gate %s: %w", gateID, err)
	}
	if report.Blocked || report.BlockCount > 0 {
		return fmt.Errorf("gate %s: %d block issue(s), first=%q in %s",
			gateID, report.BlockCount, firstBlockWord(report), firstBlockSource(report))
	}
	return nil
}

func firstBlockWord(r *artifacts.ComplianceReport) string {
	if len(r.Blocks) == 0 {
		return ""
	}
	return r.Blocks[0].Word
}

func firstBlockSource(r *artifacts.ComplianceReport) string {
	if len(r.Blocks) == 0 {
		return ""
	}
	return r.Blocks[0].Source
}

// checkLengthInRange 按 hook-plan 与目标时长校验正文字数。
func checkLengthInRange(rc *runctx.Context) error {
	path := rc.ArtifactPath("artifacts/chapter.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("gate length_in_range: %w", err)
	}

	var plan *artifacts.HookPlan
	if rc.ArtifactExists("artifacts/hook-plan.json") {
		plan, _ = artifacts.LoadHookPlan(rc.ArtifactPath("artifacts/hook-plan.json"))
	}
	minChars, maxChars := artifacts.ChapterCharBounds(rc.TargetDurationSec(), plan)
	n := artifacts.CountChapterBodyRunes(string(data))

	if n > maxChars {
		if plan != nil {
			_ = agent.EnforceChapterLength(rc, plan)
			data, _ = os.ReadFile(path)
			n = artifacts.CountChapterBodyRunes(string(data))
		}
	}
	if n < minChars || n > maxChars {
		return fmt.Errorf("gate length_in_range: chapter has %d chars (body only), want %d-%d", n, minChars, maxChars)
	}
	return nil
}

func stringsTrimDryRunComment(s string) string {
	const prefix = "<!-- dry-run -->\n"
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func checkMotionQualityOK(rc *runctx.Context) error {
	const gateID = "motion_quality_ok"
	if rc.DryRun {
		return nil
	}
	sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil {
		return fmt.Errorf("gate %s: storyboard: %w", gateID, err)
	}
	tl, err := artifacts.LoadTimeline(rc.ArtifactPath("artifacts/timeline.json"))
	if err != nil {
		return fmt.Errorf("gate %s: timeline: %w", gateID, err)
	}
	plannedKB := artifacts.CountStoryboardVisualType(sb.Shots, "ken_burns")
	actualKB := artifacts.CountVisualTypeShots(tl.Shots, "ken_burns")
	if actualKB > plannedKB {
		return fmt.Errorf("gate %s: unplanned ken_burns shots=%d (planned %d)", gateID, actualKB, plannedKB)
	}
	requireVideo := false
	if rc.App != nil && rc.App.Stack != nil {
		requireVideo = rc.App.Stack.VideoConfig().RequireVideo
	}
	if requireVideo && actualKB > 0 && plannedKB == 0 {
		return fmt.Errorf("gate %s: video-native run has %d ken_burns clips", gateID, actualKB)
	}
	timingPath := rc.ArtifactPath("artifacts/produce-timing.json")
	if data, err := os.ReadFile(timingPath); err == nil {
		if strings.Contains(string(data), "use ken burns clip") && requireVideo && plannedKB == 0 {
			return fmt.Errorf("gate %s: produce-timing reports ken burns fallback on ai_video plan", gateID)
		}
	}
	return nil
}

func checkAudioDurationOK(rc *runctx.Context) error {
	const gateID = "audio_duration_ok"
	if rc.DryRun {
		return nil
	}
	report, err := artifacts.LoadSyncReport(rc.ArtifactPath("artifacts/sync-report.json"))
	if err != nil {
		return fmt.Errorf("gate %s: %w", gateID, err)
	}
	sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil {
		return fmt.Errorf("gate %s: storyboard: %w", gateID, err)
	}
	audioSec := speechAudioSecForGate(rc, report)
	expected := sb.TotalNarrationSec
	if expected <= 0 {
		expected = sb.TotalDurationSec()
	}
	tol := 15.0
	if directorMode(rc) {
		tol = 5
	}
	if math.Abs(audioSec-expected) > tol {
		return fmt.Errorf("gate %s: audio %.1fs vs storyboard %.1fs (tol %.0fs)", gateID, audioSec, expected, tol)
	}
	return nil
}

func speechAudioSecForGate(rc *runctx.Context, report *artifacts.SyncReport) float64 {
	if report == nil {
		return 0
	}
	if report.SpeechAudioSec > 0 {
		return report.SpeechAudioSec
	}
	if rc != nil {
		if segs, err := artifacts.LoadAudioSegments(rc.ArtifactPath("artifacts/audio_segments.json")); err == nil && segs.TotalSec > 0 {
			return segs.TotalSec
		}
	}
	audioSec := report.AudioTotalSec
	if audioSec <= 0 && report.VideoTotalSec > 0 {
		audioSec = report.VideoTotalSec
	}
	return audioSec
}

func checkNarrationCompleteOK(rc *runctx.Context) error {
	const gateID = "narration_complete_ok"
	sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil {
		return fmt.Errorf("gate %s: %w", gateID, err)
	}
	if ids := sb.IncompleteNarrationShots(); len(ids) > 0 {
		return fmt.Errorf("gate %s: incomplete narration in shots %v (review storyboard before produce)", gateID, ids)
	}
	return nil
}

func checkVisualQualityOK(rc *runctx.Context) error {
	const gateID = "visual_quality_ok"
	if rc.DryRun {
		return nil
	}
	report, err := artifacts.LoadProduceDegradationReport(rc.ArtifactPath("artifacts/produce-degradation.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("gate %s: %w", gateID, err)
	}
	if report.PlannedAIVideo <= 0 {
		return nil
	}
	ratio := float64(report.DegradedCount) / float64(report.PlannedAIVideo)
	if ratio > 0.25 {
		return fmt.Errorf("gate %s: degraded %.0f%% of ai_video shots (%d/%d)", gateID, ratio*100, report.DegradedCount, report.PlannedAIVideo)
	}
	return nil
}

// EnsureRequiredArtifacts 确认必填产物文件已落盘并记入 manifest。
func EnsureRequiredArtifacts(rc *runctx.Context, stage *workflow.StageDefinition) error {
	for _, a := range stage.Artifacts.Required {
		if !rc.ArtifactExists(a.Path) {
			return fmt.Errorf("stage %q: required artifact missing: %s", stage.ID, a.Path)
		}
		rc.RecordArtifact(a.Path, a.Path, true)
	}
	for _, a := range stage.Artifacts.Optional {
		if rc.ArtifactExists(a.Path) {
			rc.RecordArtifact(a.Path, a.Path, false)
		}
	}
	return nil
}
