package artifacts

import (
	"encoding/json"
	"os"
)

// BGMPlan Produce 使用的背景音乐方案。
type BGMPlan struct {
	Mood       string  `json:"mood"`
	Tone       string  `json:"tone"`
	Source     string  `json:"source"` // library | user | none
	Path       string  `json:"path,omitempty"`
	Volume     float64 `json:"volume"`
	Reason     string  `json:"reason,omitempty"`
}

// Save 写入 bgm-plan.json。
func (p *BGMPlan) Save(path string) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadBGMPlan 读取 bgm-plan.json。
func LoadBGMPlan(path string) (*BGMPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p BGMPlan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
