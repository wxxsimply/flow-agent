package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// DeleteScenesForRewrite 删除指定场景 part 文件，便于 continuity 回退后重写。
func DeleteScenesForRewrite(rc *runctx.Context, sceneIDs []int) error {
	for _, id := range sceneIDs {
		rel := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", id))
		path := rc.ArtifactPath(rel)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// WriteRewriteHints 将 continuity 问题写入重写提示文件，供 Writer 读取。
func WriteRewriteHints(rc *runctx.Context, report *artifacts.ContinuityReport) error {
	hints := &artifacts.RewriteHints{Scenes: map[string]string{}}
	for _, iss := range report.Issues {
		if iss.Severity != "critical" || iss.SceneID <= 0 {
			continue
		}
		key := fmt.Sprintf("%d", iss.SceneID)
		block := iss.Message
		if iss.Suggestion != "" {
			block += "\n修改建议: " + iss.Suggestion
		}
		if prev, ok := hints.Scenes[key]; ok {
			hints.Scenes[key] = prev + "\n\n" + block
		} else {
			hints.Scenes[key] = block
		}
		// 若建议涉及下一幕，复制一份到下一幕 key
		combined := iss.Message + " " + iss.Suggestion
		if artifacts.MentionsNextScene(combined) {
			nextKey := fmt.Sprintf("%d", iss.SceneID+1)
			if _, ok := hints.Scenes[nextKey]; !ok {
				hints.Scenes[nextKey] = "承接上一幕修正，注意: " + block
			}
		}
	}
	if len(hints.Scenes) == 0 {
		return nil
	}
	b, err := json.MarshalIndent(hints, "", "  ")
	if err != nil {
		return err
	}
	return rc.WriteArtifact(artifacts.RewriteHintsRel, b)
}

// ClearRewriteHints 删除重写提示文件。
func ClearRewriteHints(rc *runctx.Context) {
	_ = os.Remove(rc.ArtifactPath(artifacts.RewriteHintsRel))
}

// RewriteScenesAfterContinuity 根据 continuity 报告删除 critical 场景并带修复说明重写。
func RewriteScenesAfterContinuity(rc *runctx.Context, report *artifacts.ContinuityReport, plan *artifacts.HookPlan) error {
	valid := map[int]bool{}
	for _, s := range plan.Scenes {
		valid[s.ID] = true
	}
	ids := report.SceneIDsForRewrite(valid)
	if len(ids) == 0 {
		return nil
	}
	if err := WriteRewriteHints(rc, report); err != nil {
		return err
	}
	if err := DeleteScenesForRewrite(rc, ids); err != nil {
		return err
	}
	if err := RunWriter(rc); err != nil {
		return err
	}
	ClearRewriteHints(rc)
	return nil
}

// FullRewriteAfterContinuity 清空全部场景并按 continuity 意见整章重写（最后一次重试）。
func FullRewriteAfterContinuity(rc *runctx.Context, report *artifacts.ContinuityReport, plan *artifacts.HookPlan) error {
	if err := ClearChapterDraft(rc); err != nil {
		return err
	}
	if err := WriteRewriteHints(rc, report); err != nil {
		return err
	}
	if err := RunWriter(rc); err != nil {
		return err
	}
	ClearRewriteHints(rc)
	return nil
}
