package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// ClearChapterDraft 删除本章草稿与 continuity 产物（保留 hook-plan / brief）。
func ClearChapterDraft(rc *runctx.Context) error {
	for _, rel := range []string{
		"artifacts/chapter.md",
		"artifacts/continuity-report.json",
		artifacts.RewriteHintsRel,
		"artifacts/character-state.patch.json",
	} {
		if err := os.Remove(rc.ArtifactPath(rel)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	partsDir := rc.ArtifactPath("artifacts/chapter.parts")
	entries, err := os.ReadDir(partsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(partsDir, e.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// ClearAssembleArtifacts 删除 assemble 及后续阶段的产物，便于 iterate 重跑。
func ClearAssembleArtifacts(rc *runctx.Context) error {
	for _, rel := range []string{
		"artifacts/shot-language-brief.md",
		"artifacts/shot-language-expand.json",
		"artifacts/storyboard.json",
		"artifacts/storyboard-review.json",
		"artifacts/narration.ssml",
		"artifacts/continuity-ledger.md",
		"artifacts/character-sheets.json",
		"artifacts/prop-sheets.json",
		"artifacts/applied-skills.json",
		"artifacts/master.mp4",
		"artifacts/cover.jpg",
		"artifacts/publish-pack.json",
		"artifacts/timeline.json",
	} {
		if err := os.Remove(rc.ArtifactPath(rel)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	assetsDir := rc.ArtifactPath("artifacts/assets")
	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		p := filepath.Join(assetsDir, e.Name())
		if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// PrepareResumeFromStage 在 resume 进入某阶段前清理过期产物。
func PrepareResumeFromStage(rc *runctx.Context, fromStage string, keepChapter bool) error {
	switch fromStage {
	case "assemble":
		if err := ClearAssembleArtifacts(rc); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "resume: cleared assemble artifacts for re-run")
	case "write":
		if keepChapter {
			return nil
		}
		if err := ClearChapterDraft(rc); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "resume: cleared chapter draft (use --keep-chapter to retain existing parts)")
	case "continuity":
		for _, rel := range []string{
			"artifacts/continuity-report.json",
			artifacts.RewriteHintsRel,
			"artifacts/character-state.patch.json",
		} {
			_ = os.Remove(rc.ArtifactPath(rel))
		}
	}
	return nil
}
