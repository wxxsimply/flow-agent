package artifacts

import (
	"encoding/json"
	"os"
)

// BoNCandidateScore 单条 BoN 候选评分。
type BoNCandidateScore struct {
	Index      int     `json:"index"`
	Score      float64 `json:"score"`
	Path       string  `json:"path,omitempty"`
	Unreliable bool    `json:"unreliable,omitempty"`
}

// BoNScoreEntry 单段 i2v BoN 选优记录。
type BoNScoreEntry struct {
	ShotID             string              `json:"shot_id,omitempty"`
	ClipPath           string              `json:"clip_path"`
	Scorer             string              `json:"scorer"`
	Candidates         []BoNCandidateScore `json:"candidates"`
	Selected           int                 `json:"selected"`
	SelectedScore      float64             `json:"selected_score"`
	SelectedUnreliable bool                `json:"selected_unreliable,omitempty"`
}

// ProduceDegradationReport 记录 produce 阶段 i2v 降级与 API 错误摘要。
type ProduceDegradationReport struct {
	KenBurnsShots   []string        `json:"ken_burns_shots,omitempty"`
	DegradedCount   int             `json:"degraded_count"`
	PlannedAIVideo  int             `json:"planned_ai_video"`
	ActualKenBurns  int             `json:"actual_ken_burns"`
	APIErrorSamples []string        `json:"api_error_samples,omitempty"`
	WMRewardBoN     []BoNScoreEntry `json:"wmreward_bon,omitempty"`
}

// Save 写入 produce-degradation.json。
func (r *ProduceDegradationReport) Save(path string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadProduceDegradationReport 读取 produce-degradation.json。
func LoadProduceDegradationReport(path string) (*ProduceDegradationReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r ProduceDegradationReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// CountVisualType 统计 storyboard/timeline 中某 visual_type 镜数。
func CountVisualTypeShots(shots []TimelineShot, visualType string) int {
	n := 0
	for _, s := range shots {
		if s.VisualType == visualType {
			n++
		}
	}
	return n
}

// CountStoryboardVisualType 统计 storyboard 镜数。
func CountStoryboardVisualType(shots []Shot, visualType string) int {
	n := 0
	for _, s := range shots {
		if s.VisualType == visualType {
			n++
		}
	}
	return n
}
