package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/agent/skills"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunShotAssembler director 模式：第一镜文本 → 约 3000 字扩写 → 自动生成全部分镜 → storyboard.json。
func RunShotAssembler(rc *runctx.Context) error {
	exp, err := RunShotLanguageExpander(rc)
	if err != nil {
		return err
	}
	sb, err := expandedToStoryboard(rc, exp)
	if err != nil {
		return err
	}
	applyStoryboardProduceCaps(rc, sb)

	if err := writeAppliedSkillsArtifact(rc); err != nil {
		return err
	}

	report, err := applyStoryboardPostProcess(rc, sb)
	if err != nil {
		return err
	}
	if err := artifacts.SaveStoryboardReview(rc.ArtifactPath("artifacts/storyboard-review.json"), *report); err != nil {
		return err
	}
	if len(report.Issues) > 0 {
		slog.Info("storyboard review", "issues", len(report.Issues), "auto_fixed", report.Fixed)
	}
	rc.RecordArtifact("storyboard-review.json", "artifacts/storyboard-review.json", false)

	brief := ""
	if data, err := os.ReadFile(rc.ArtifactPath("artifacts/shot-language-brief.md")); err == nil {
		brief = string(data)
	}
	ledger := artifacts.BuildContinuityLedger(sb, brief, nil)
	if err := rc.WriteArtifact("artifacts/continuity-ledger.md", []byte(ledger)); err != nil {
		return err
	}
	rc.RecordArtifact("continuity-ledger.md", "artifacts/continuity-ledger.md", false)

	sheetsPath := rc.ArtifactPath("artifacts/character-sheets.json")
	if rc.App != nil && rc.App.Stack.AssembleConfig().CharacterTurnaround {
		if err := RunAssembleProtagonistTurnaround(rc, sb); err != nil {
			return fmt.Errorf("protagonist turnaround: %w", err)
		}
	} else if light := artifacts.BuildLightCharacterSheets(sb); light != nil {
		if err := light.Save(sheetsPath); err != nil {
			return err
		}
		rc.RecordArtifact("character-sheets.json", "artifacts/character-sheets.json", false)
	}

	var propSheets *artifacts.PropSheets
	if rc.App != nil && rc.App.Stack.AssembleConfig().PropTurnaround {
		if err := RunPropDesigner(rc, sb); err != nil {
			return fmt.Errorf("prop turnaround: %w", err)
		}
		if loaded, err := artifacts.LoadPropSheets(rc.ArtifactPath("artifacts/prop-sheets.json")); err == nil {
			propSheets = loaded
		}
		ledger := artifacts.BuildContinuityLedger(sb, brief, propSheets)
		if err := rc.WriteArtifact("artifacts/continuity-ledger.md", []byte(ledger)); err != nil {
			return err
		}
	}
	pol := artifacts.DirectorPolicy()
	if err := sb.Validate(rc.EpisodeNo, rc.TargetDurationSec(), pol); err != nil {
		return fmt.Errorf("assemble storyboard: %w", err)
	}

	data, err := json.MarshalIndent(sb, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/storyboard.json", data); err != nil {
		return err
	}
	ssml := buildNarrationSSMLFromShots(sb.Shots)
	if err := rc.WriteArtifact("artifacts/narration.ssml", []byte(ssml)); err != nil {
		return err
	}
	rc.RecordArtifact("storyboard.json", "artifacts/storyboard.json", true)
	rc.RecordArtifact("narration.ssml", "artifacts/narration.ssml", true)
	return nil
}

func loadUserShots(rc *runctx.Context) ([]artifacts.UserShotInput, error) {
	if rc.Creative != nil && len(rc.Creative.Shots) > 0 {
		return rc.Creative.Shots, nil
	}
	path := rc.ArtifactPath("artifacts/user-shots.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load user shots: %w", err)
	}
	var list []artifacts.UserShotInput
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func writeAppliedSkillsArtifact(rc *runctx.Context) error {
	rep := skills.Report()
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/applied-skills.json", data); err != nil {
		return err
	}
	rc.RecordArtifact("applied-skills.json", "artifacts/applied-skills.json", false)
	return nil
}

func buildNarrationSSMLFromShots(shots []artifacts.Shot) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><speak>`)
	for _, s := range shots {
		b.WriteString("<p>")
		b.WriteString(escapeSSML(s.Narration))
		b.WriteString("</p>")
	}
	b.WriteString("</speak>")
	return b.String()
}
