package agent

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

var propTurnaroundViewSpecs = []struct {
	key   string
	label string
}{
	{"front", "正面，面朝镜头"},
	{"side", "侧面，90度侧视"},
	{"back", "背面，背对镜头"},
}

// RunPropDesigner 从分镜提取物体、生成三视图、写 prop-sheets.json 并更新 storyboard prop_refs。
func RunPropDesigner(rc *runctx.Context, sb *artifacts.Storyboard) error {
	if sb == nil || len(sb.Shots) == 0 {
		slog.Warn("empty storyboard, skip prop designer")
		return nil
	}
	sheets := artifacts.CollectPropsFromStoryboard(sb)
	if sheets == nil || len(sheets.Props) == 0 {
		slog.Info("no props to design, skip prop designer")
		return nil
	}

	outDir := rc.ArtifactPath(filepath.Join("artifacts", "assets", "props"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	preset := VisualPresetForRun(rc)
	LoadCreativeOptionsFromRun(rc)
	spec := config.MediaSpecFromCreative(rc.Creative)

	for i := range sheets.Props {
		pe := &sheets.Props[i]
		stem := pe.ID
		views := &artifacts.CharacterViewPaths{}
		var prompts []string

		for _, vs := range propTurnaroundViewSpecs {
			rel := filepath.ToSlash(filepath.Join("artifacts", "assets", "props", stem+"-"+vs.key+".png"))
			abs := rc.ArtifactPath(rel)
			prompt := strings.TrimSpace(preset.ImageStylePrefix) + "\n" +
				fmt.Sprintf("物体设定图，%s，同一物体外观完全一致，白色纯背景，无人物，清晰线条与分层，单物体单视角。\n物体名：%s\n外观锁定：%s\n构图：%s。",
					vs.label, pe.Name, pe.Appearance, spec.LabelZH) +
				" [NEG] " + preset.ImageNegative + "，多视角拼图，三视图拼在一张图，人物，人手"

			prompts = append(prompts, prompt)
			if err := writeTurnaroundImage(rc, abs, prompt, spec, pe.Name+" "+vs.label); err != nil {
				return err
			}
			switch vs.key {
			case "front":
				views.Front = rel
			case "side":
				views.Side = rel
			case "back":
				views.Back = rel
			}
		}
		pe.TurnaroundPath = views.Front
		pe.TurnaroundViews = views
		_ = prompts
	}

	artifacts.AlignHeldPropsToRegistry(sb, sheets)
	artifacts.ApplyPropRefs(sb, sheets)

	if review := sb.ReviewStoryboardWithProps(sheets); len(review.Issues) > 0 {
		slog.Info("prop continuity review", "issues", len(review.Issues))
	}

	if err := sheets.Save(rc.ArtifactPath("artifacts/prop-sheets.json")); err != nil {
		return err
	}
	rc.RecordArtifact("prop-sheets.json", "artifacts/prop-sheets.json", false)
	rc.RecordArtifact("assets/props/", "artifacts/assets/props/", false)
	return nil
}

// RunPropDesignerFromArtifacts auto 模式：从已有 storyboard.json 加载并执行。
func RunPropDesignerFromArtifacts(rc *runctx.Context) error {
	sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json"))
	if err != nil {
		return fmt.Errorf("load storyboard: %w", err)
	}
	if err := RunPropDesigner(rc, sb); err != nil {
		return err
	}
	if err := sb.Save(rc.ArtifactPath("artifacts/storyboard.json")); err != nil {
		return err
	}
	rc.RecordArtifact("storyboard.json", "artifacts/storyboard.json", true)
	return nil
}
