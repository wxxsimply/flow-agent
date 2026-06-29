package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/compose/bgm"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// SaveCreativeOptions 落盘用户创作参数。
func SaveCreativeOptions(rc *runctx.Context) error {
	if rc.Creative == nil {
		return nil
	}
	rc.Creative.Normalize()
	data, err := json.MarshalIndent(rc.Creative, "", "  ")
	if err != nil {
		return err
	}
	return rc.WriteArtifact("artifacts/creative-options.json", data)
}

// LoadCreativeOptionsFromRun 续跑时恢复创作参数。
func LoadCreativeOptionsFromRun(rc *runctx.Context) {
	if rc.Creative != nil {
		return
	}
	path := rc.ArtifactPath("artifacts/creative-options.json")
	c, err := artifacts.LoadCreativeOptions(path)
	if err == nil {
		rc.Creative = c
		return
	}
	rc.Creative = &artifacts.CreativeOptions{
		InputMode: "auto", AnimationStyle: "2d", BGMMode: "auto",
		NarratorVoice: "epic_male", TargetDurationSec: 150,
	}
	rc.Creative.Normalize()
}

// ResolveNarratorForRun 解析旁白音色与语速。
func ResolveNarratorForRun(rc *runctx.Context, ttsCfg config.StackTTSConfig) (voice string, speed float64) {
	LoadCreativeOptionsFromRun(rc)
	voiceID := "epic_male"
	if rc.Creative != nil && rc.Creative.NarratorVoice != "" {
		voiceID = rc.Creative.NarratorVoice
	}
	voice, speed = config.ResolveNarratorVoice(voiceID, ttsCfg.Voice)
	if isMicroMovieStack(rc) && speed < 0.95 {
		speed = 0.95
	}
	return voice, speed
}

// VisualPresetForRun 当前运行的 2D/3D 与画面约束。
func VisualPresetForRun(rc *runctx.Context) config.VisualStylePreset {
	LoadCreativeOptionsFromRun(rc)
	return config.VisualPresetFromCreative(rc.Creative)
}

// BuildBGMPlan 根据 story-spine 情绪选择曲库 BGM。
func BuildBGMPlan(rc *runctx.Context, spine *artifacts.StorySpine) (*artifacts.BGMPlan, error) {
	LoadCreativeOptionsFromRun(rc)
	plan := &artifacts.BGMPlan{Volume: 0.22, Source: "none"}
	if rc.App != nil && rc.App.Stack != nil {
		if v := rc.App.Stack.ComposeConfig().BGMVolume; v > 0 {
			plan.Volume = v
		}
	}
	if rc.Creative != nil && rc.Creative.BGMMode == "off" {
		plan.Reason = "user disabled bgm"
		return plan, nil
	}
	if rc.Creative != nil && rc.Creative.BGMMode == "path" && rc.Creative.BGMPath != "" {
		p := rc.Creative.BGMPath
		if !filepath.IsAbs(p) && rc.App != nil {
			p = filepath.Join(rc.App.Root, p)
		}
		if fileExistsAgent(p) {
			plan.Source = "user"
			plan.Path = p
			plan.Reason = "user bgm path"
			return plan, nil
		}
	}

	mood, tone := "", ""
	if spine != nil {
		mood = spine.Mood
		tone = spine.Tone
	}
	libDir := filepath.Join(rc.App.Root, "assets", "bgm")
	if rc.App != nil && rc.App.Stack != nil {
		if p := strings.TrimSpace(rc.App.Stack.ComposeConfig().BGMLibrary); p != "" {
			libDir = p
			if !filepath.IsAbs(libDir) {
				libDir = filepath.Join(rc.App.Root, libDir)
			}
		}
	}
	path, resolved, ok := bgm.SelectPath(libDir, mood, tone)
	if !ok {
		plan.Reason = "no matching file in assets/bgm/ (add " + string(resolved) + ".mp3)"
		return plan, nil
	}
	plan.Source = "library"
	plan.Path = path
	plan.Mood = string(resolved)
	plan.Tone = tone
	plan.Reason = "matched mood from story-spine"
	return plan, nil
}

// PersistBGMPlan 写入并记录 bgm-plan。
func PersistBGMPlan(rc *runctx.Context, plan *artifacts.BGMPlan) error {
	if plan == nil {
		return nil
	}
	path := rc.ArtifactPath("artifacts/bgm-plan.json")
	if err := plan.Save(path); err != nil {
		return err
	}
	rc.RecordArtifact("bgm-plan.json", "artifacts/bgm-plan.json", true)
	return nil
}

// ResolveBGMPathForCompose 供 produce 合成使用。
func ResolveBGMPathForCompose(rc *runctx.Context) string {
	planPath := rc.ArtifactPath("artifacts/bgm-plan.json")
	if plan, err := artifacts.LoadBGMPlan(planPath); err == nil && plan.Path != "" {
		return plan.Path
	}
	return ""
}

func fileExistsAgent(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
