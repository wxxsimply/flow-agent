package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// PublishMetrics 单集发布后的播放/互动数据（series publish-metrics 与 run snapshot 共用）。
type PublishMetrics struct {
	EpisodeNo        int       `json:"episode_no"`
	SeriesID         string    `json:"series_id,omitempty"`
	Platform         string    `json:"platform,omitempty"`
	Views24h         int64     `json:"views_24h"`
	CompletionRate   float64   `json:"completion_rate"`
	Likes            int64     `json:"likes,omitempty"`
	CommentKeywords  []string  `json:"comment_keywords,omitempty"`
	Note             string    `json:"note,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Normalize 填充默认值并校验范围。
func (m *PublishMetrics) Normalize() error {
	if m.EpisodeNo <= 0 {
		return fmt.Errorf("episode_no required")
	}
	if m.Platform == "" {
		m.Platform = "douyin"
	}
	if m.CompletionRate < 0 {
		m.CompletionRate = 0
	}
	if m.CompletionRate > 1 {
		return fmt.Errorf("completion_rate must be 0-1")
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = time.Now().UTC()
	}
	return nil
}

// FormatForPlanner 供 Planner prompt 引用的简短文本。
func (m *PublishMetrics) FormatForPlanner() string {
	if m == nil {
		return ""
	}
	kw := strings.Join(m.CommentKeywords, "、")
	if kw == "" {
		kw = "（暂无）"
	}
	return fmt.Sprintf(`上集（第 %d 集）发布数据 [%s]:
- 24h 播放: %d
- 完播率: %.1f%%
- 点赞: %d
- 评论热词: %s
- 备注: %s`,
		m.EpisodeNo, m.Platform, m.Views24h, m.CompletionRate*100, m.Likes, kw, orNote(m.Note))
}

func orNote(s string) string {
	if strings.TrimSpace(s) == "" {
		return "无"
	}
	return s
}

// LoadPublishMetrics 读取 JSON 文件。
func LoadPublishMetrics(path string) (*PublishMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m PublishMetrics
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save 写入 JSON 文件。
func (m *PublishMetrics) Save(path string) error {
	if err := m.Normalize(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadMetricsSnapshot 读取 run 内 metrics-snapshot.json。
func LoadMetricsSnapshot(path string) (*PublishMetrics, error) {
	return LoadPublishMetrics(path)
}
