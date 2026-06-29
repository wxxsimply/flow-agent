package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/image"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

var turnaroundViewSpecs = []struct {
	key   string
	label string
}{
	{"front", "正面全身站立，面朝镜头"},
	{"side", "侧面全身站立，90度侧视"},
	{"back", "背面全身站立，背对镜头"},
}

// RunCharacterDesigner 为每个角色生成三视图设定图（正/侧/背各一张），并写入 character-sheets.json。
func RunCharacterDesigner(rc *runctx.Context) error {
	spine, err := artifacts.LoadStorySpine(rc.ArtifactPath("artifacts/story-spine.json"))
	if err != nil {
		return fmt.Errorf("load story-spine: %w", err)
	}
	if spine == nil || len(spine.Characters) == 0 {
		slog.Warn("no characters in story-spine, skip character designer")
		return nil
	}

	outDir := rc.ArtifactPath(filepath.Join("artifacts", "assets", "characters"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	sheets := &artifacts.CharacterSheets{}
	preset := VisualPresetForRun(rc)
	LoadCreativeOptionsFromRun(rc)
	spec := config.MediaSpecFromCreative(rc.Creative)

	for i, ch := range spine.Characters {
		name := strings.TrimSpace(ch.Name)
		app := strings.TrimSpace(ch.Appearance)
		if name == "" || app == "" {
			continue
		}
		id := fmt.Sprintf("c%02d", i+1)
		stem := id + "-" + safeFileStem(name)
		views := &artifacts.CharacterViewPaths{}
		var prompts []string

		for _, vs := range turnaroundViewSpecs {
			rel := filepath.ToSlash(filepath.Join("artifacts", "assets", "characters", stem+"-"+vs.key+".png"))
			abs := rc.ArtifactPath(rel)
			prompt := strings.TrimSpace(preset.ImageStylePrefix) + "\n" +
				fmt.Sprintf("角色设定图，%s，同一角色外观完全一致，白色纯背景，全身，清晰线条与分层，单角色单视角。\n角色名：%s\n外观锁定：%s\n构图：%s。",
					vs.label, name, app, spec.LabelZH) +
				" [NEG] " + preset.ImageNegative + "，多视角拼图，三视图拼在一张图"

			prompts = append(prompts, prompt)
			if err := writeTurnaroundImage(rc, abs, prompt, spec, name+" "+vs.label); err != nil {
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

		frontRel := views.Front
		sheets.Characters = append(sheets.Characters, artifacts.CharacterSheetEntry{
			Name:             name,
			Appearance:       app,
			TurnaroundPath:   frontRel,
			TurnaroundPrompt: strings.Join(prompts, "\n---\n"),
			TurnaroundViews:  views,
		})
	}

	if err := sheets.Save(rc.ArtifactPath("artifacts/character-sheets.json")); err != nil {
		return err
	}
	rc.RecordArtifact("character-sheets.json", "artifacts/character-sheets.json", false)
	rc.RecordArtifact("assets/characters/", "artifacts/assets/characters/", false)
	return nil
}

func writeTurnaroundImage(rc *runctx.Context, abs, prompt string, spec config.MediaSpec, label string) error {
	if rc.DryRun {
		return ffmpeg.GeneratePlaceholderPNG(abs, "turnaround "+label)
	}
	if rc.Providers == nil || rc.Providers.Image == nil {
		return fmt.Errorf("image provider not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	imgCfg := rc.App.Stack.ImageConfig()
	imgCfg = config.ApplyMediaSpec(imgCfg, spec)
	imgBytes, genErr := generateImageWithFallback(ctx, rc, imgCfg, image.GenerateRequest{
		Prompt:      prompt,
		AspectRatio: spec.AspectRatio,
		Width:       spec.Width,
		Height:      spec.Height,
	})
	cancel()
	if genErr != nil {
		slog.Warn("turnaround generate failed, placeholder", "label", label, "err", genErr)
		return ffmpeg.GeneratePlaceholderPNG(abs, "turnaround "+label)
	}
	return os.WriteFile(abs, imgBytes, 0o644)
}

func safeFileStem(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	if s == "" {
		return "character"
	}
	return s
}
