// Package artifacts 定义跨阶段共享的产物与 manifest 数据结构。
package artifacts

import (
	"encoding/json"
	"os"
	"time"
)

// Manifest 单次运行的元数据，对应 runs/<id>/manifest.json。
type Manifest struct {
	RunID      string          `json:"run_id"`
	TraceID    string          `json:"trace_id"`
	Workflow   string          `json:"workflow"`
	SeriesID   string          `json:"series_id"`
	EpisodeNo  int             `json:"episode_no"`
	Stage      string          `json:"stage"`
	Stack      string          `json:"stack_profile,omitempty"`
	Gates      map[string]bool `json:"gates"`
	Artifacts  []ArtifactEntry `json:"artifacts"`
	Cost       *CostLedger     `json:"cost,omitempty"`
	StartedAt  time.Time       `json:"started_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
	FinishedAt *time.Time      `json:"finished_at,omitempty"`
	DryRun     bool            `json:"dry_run,omitempty"`
	LastError  string          `json:"last_error,omitempty"`
	UserID     string          `json:"user_id,omitempty"`
	Title      string          `json:"title,omitempty"`
	RunDir     string          `json:"run_dir,omitempty"`
}

// ArtifactEntry manifest 中登记的一个产物。
type ArtifactEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Required bool   `json:"required"`
}

// CostLedger 分项成本（人民币），对应 cost-ledger.json。
type CostLedger struct {
	LLMCNY              float64 `json:"llm_cny"`
	TTSCNY              float64 `json:"tts_cny"`
	ImageCNY            float64 `json:"image_cny"`
	VideoCNY            float64 `json:"video_cny"`
	OtherCNY            float64 `json:"other_cny"`
	TotalCNY            float64 `json:"total_cny"`
	LLMPromptTokens     int     `json:"llm_prompt_tokens,omitempty"`
	LLMCompletionTokens int     `json:"llm_completion_tokens,omitempty"`
	TTSCharacters       int     `json:"tts_characters,omitempty"`
	ImageCount          int     `json:"image_count,omitempty"`
	VideoSeconds        float64 `json:"video_seconds,omitempty"`
	VideoAPICalls       int     `json:"video_api_calls,omitempty"`
}

// Recalc 重算 TotalCNY。
func (c *CostLedger) Recalc() {
	c.TotalCNY = c.LLMCNY + c.TTSCNY + c.ImageCNY + c.VideoCNY + c.OtherCNY
}

// LoadManifest 从磁盘读取 manifest。
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Save 写入 manifest 并更新 UpdatedAt。
func (m *Manifest) Save(path string) error {
	m.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
