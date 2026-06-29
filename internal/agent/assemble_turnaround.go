package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunAssembleProtagonistTurnaround 为 director assemble 生成主角正面 turnaround（单视图）。
func RunAssembleProtagonistTurnaround(rc *runctx.Context, sb *artifacts.Storyboard) error {
	if rc == nil || sb == nil || len(sb.Shots) == 0 {
		return nil
	}
	light := artifacts.BuildLightCharacterSheets(sb)
	if light == nil || len(light.Characters) == 0 {
		return nil
	}
	ch := light.Characters[0]
	name := strings.TrimSpace(ch.Name)
	if name == "" {
		name = "主角"
	}
	app := strings.TrimSpace(ch.Appearance)

	outDir := rc.ArtifactPath(filepath.Join("artifacts", "assets", "characters"))
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	stem := "c01-protagonist"
	rel := filepath.ToSlash(filepath.Join("artifacts", "assets", "characters", stem+"-front.png"))
	abs := rc.ArtifactPath(rel)

	LoadCreativeOptionsFromRun(rc)
	spec := config.MediaSpecFromCreative(rc.Creative)
	preset := VisualPresetForRun(rc)
	prompt := strings.TrimSpace(preset.ImageStylePrefix) + "\n" +
		fmt.Sprintf("角色设定图，正面全身站立，面朝镜头，同一角色外观完全一致，白色纯背景，全身，清晰线条与分层，单角色单视角。\n角色名：%s\n外观锁定：%s\n构图：%s。",
			name, app, spec.LabelZH) +
		" [NEG] " + preset.ImageNegative + "，多视角拼图，三视图拼在一张图"

	if err := writeTurnaroundImage(rc, abs, prompt, spec, name+" front"); err != nil {
		return err
	}

	sheets := &artifacts.CharacterSheets{
		Characters: []artifacts.CharacterSheetEntry{{
			Name:             name,
			Appearance:       app,
			TurnaroundPath:   rel,
			TurnaroundViews:  &artifacts.CharacterViewPaths{Front: rel},
			TurnaroundPrompt: prompt,
		}},
	}
	path := rc.ArtifactPath("artifacts/character-sheets.json")
	if err := sheets.Save(path); err != nil {
		return err
	}
	rc.RecordArtifact("character-sheets.json", "artifacts/character-sheets.json", false)
	rc.RecordArtifact("assets/characters/", "artifacts/assets/characters/", false)
	return nil
}

func turnaroundFrontSeedPath(rc *runctx.Context) string {
	if rc == nil {
		return ""
	}
	sheets, err := artifacts.LoadCharacterSheets(rc.ArtifactPath("artifacts/character-sheets.json"))
	if err != nil || sheets == nil || len(sheets.Characters) == 0 {
		return ""
	}
	ch := sheets.Characters[0]
	if ch.TurnaroundViews != nil && ch.TurnaroundViews.Front != "" {
		return rc.ArtifactPath(ch.TurnaroundViews.Front)
	}
	if ch.TurnaroundPath != "" {
		return rc.ArtifactPath(ch.TurnaroundPath)
	}
	return ""
}

func seedFirstShotFromTurnaround(rc *runctx.Context, imgCfg config.StackImageConfig, assetsDir string) {
	if rc == nil || !imgCfg.UseTurnaroundSeed {
		return
	}
	front := turnaroundFrontSeedPath(rc)
	if front == "" {
		return
	}
	dst := filepath.Join(assetsDir, "s01.png")
	if _, err := os.Stat(dst); err == nil {
		return
	}
	data, err := os.ReadFile(front)
	if err != nil || len(data) == 0 {
		return
	}
	_ = os.WriteFile(dst, data, 0o644)
}
