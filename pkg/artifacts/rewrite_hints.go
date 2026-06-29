package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
)

// RewriteHints continuity 回退重写时，按场景注入的修复说明。
type RewriteHints struct {
	Scenes map[string]string `json:"scenes"`
}

const RewriteHintsRel = "artifacts/continuity-rewrite-hints.json"

// LoadRewriteHints 读取重写提示；不存在时返回 nil。
func LoadRewriteHints(path string) (*RewriteHints, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var h RewriteHints
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// HintForScene 返回某场景的修复说明。
func (h *RewriteHints) HintForScene(sceneID int) string {
	if h == nil || h.Scenes == nil {
		return ""
	}
	return h.Scenes[formatSceneKey(sceneID)]
}

func formatSceneKey(id int) string {
	return fmt.Sprintf("%d", id)
}
