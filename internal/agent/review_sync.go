package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// ExtractBriefBody 从审阅正文或 shot-language-brief.md 内容中提取可编辑正文（去掉 header）。
func ExtractBriefBody(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, "\n---\n"); idx >= 0 {
		return strings.TrimSpace(raw[idx+len("\n---\n"):])
	}
	if strings.HasPrefix(raw, "# ") {
		parts := strings.SplitN(raw, "\n\n", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[1], "# ") {
			rest := parts[1]
			if j := strings.Index(rest, "\n---\n"); j >= 0 {
				return strings.TrimSpace(rest[j+len("\n---\n"):])
			}
		}
	}
	return raw
}

// SyncReviewArtifacts 审阅保存：同步 expand.json、brief.md、storyboard.json、narration.ssml。
func SyncReviewArtifacts(rc *runctx.Context, briefBody string, exp *artifacts.ShotLanguageExpand, sb *artifacts.Storyboard) error {
	if rc == nil {
		return fmt.Errorf("nil run context")
	}
	if exp == nil {
		path := rc.ArtifactPath("artifacts/shot-language-expand.json")
		loaded, err := artifacts.LoadShotLanguageExpand(path)
		if err != nil {
			return fmt.Errorf("load shot-language-expand.json: %w", err)
		}
		exp = loaded
	}
	if strings.TrimSpace(briefBody) != "" {
		exp.ShotLanguageBrief = ExtractBriefBody(briefBody)
	}
	opening := strings.TrimSpace(exp.OpeningShot)
	if opening == "" {
		if data, err := os.ReadFile(rc.ArtifactPath("artifacts/opening-shot-input.md")); err == nil {
			opening = strings.TrimSpace(string(data))
			exp.OpeningShot = opening
		}
	}
	if sb != nil && len(sb.Shots) > 0 {
		mergeStoryboardIntoExpand(exp, sb)
	}
	var err error
	sb, err = expandedToStoryboard(rc, exp)
	if err != nil {
		return err
	}
	if existing, err := loadStoryboardFile(rc); err == nil && existing != nil {
		sb.EpisodeNo = existing.EpisodeNo
		sb.TargetDurationSec = existing.TargetDurationSec
	}
	applyStoryboardProduceCaps(rc, sb)
	sb.SyncTotalNarrationSec()

	if err := exp.Save(rc.ArtifactPath("artifacts/shot-language-expand.json")); err != nil {
		return err
	}
	if err := persistShotLanguageExpand(rc, exp, opening); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sb, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/storyboard.json", data); err != nil {
		return err
	}
	rc.RecordArtifact("storyboard.json", "artifacts/storyboard.json", true)
	ssml := buildNarrationSSMLFromShots(sb.Shots)
	if err := rc.WriteArtifact("artifacts/narration.ssml", []byte(ssml)); err != nil {
		return err
	}
	rc.RecordArtifact("narration.ssml", "artifacts/narration.ssml", true)
	return nil
}

func loadStoryboardFile(rc *runctx.Context) (*artifacts.Storyboard, error) {
	data, err := os.ReadFile(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil {
		return nil, err
	}
	var sb artifacts.Storyboard
	if err := json.Unmarshal(data, &sb); err != nil {
		return nil, err
	}
	return &sb, nil
}

func mergeStoryboardIntoExpand(exp *artifacts.ShotLanguageExpand, sb *artifacts.Storyboard) {
	if exp == nil || sb == nil {
		return
	}
	byID := map[string]artifacts.Shot{}
	for _, s := range sb.Shots {
		id := strings.TrimSpace(s.ID)
		if id != "" {
			byID[id] = s
		}
	}
	for i := range exp.Shots {
		es := &exp.Shots[i]
		st, ok := byID[strings.TrimSpace(es.ID)]
		if !ok {
			continue
		}
		if v := strings.TrimSpace(st.Narration); v != "" {
			es.Narration = v
		}
		if v := strings.TrimSpace(st.VisualPrompt); v != "" {
			es.VisualPrompt = v
		}
		if v := strings.TrimSpace(st.SceneBackground); v != "" {
			es.SceneBackground = v
		}
		if len(st.ActionBeats) > 0 {
			es.ActionBeats = st.ActionBeats
		}
		if v := strings.TrimSpace(st.PhysicsCues); v != "" {
			es.PhysicsCues = v
		}
		if v := strings.TrimSpace(st.ForbiddenPhysics); v != "" {
			es.ForbiddenPhysics = v
		}
		if st.DurationSec > 0 {
			es.DurationSec = st.DurationSec
		}
		if v := strings.TrimSpace(st.ShotSize); v != "" {
			es.ShotSize = v
		}
		if v := strings.TrimSpace(st.CameraAngle); v != "" {
			es.CameraAngle = v
		}
		if v := strings.TrimSpace(st.NarrativeBeat); v != "" {
			es.NarrativeBeat = v
		}
		if v := strings.TrimSpace(st.BriefExcerpt); v != "" {
			es.BriefExcerpt = v
		}
	}
}
