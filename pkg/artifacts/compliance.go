package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ComplianceIssue 单条合规命中。
type ComplianceIssue struct {
	Severity string `json:"severity"` // block | warning
	Word     string `json:"word"`
	Source   string `json:"source"`
	Snippet  string `json:"snippet,omitempty"`
}

// ComplianceReport compliance_report_v1。
type ComplianceReport struct {
	EpisodeNo    int               `json:"episode_no"`
	Blocked      bool              `json:"blocked"`
	BlockCount   int               `json:"block_count"`
	WarningCount int               `json:"warning_count"`
	Blocks       []ComplianceIssue `json:"blocks,omitempty"`
	Warnings     []ComplianceIssue `json:"warnings"`
	CheckedAt    string            `json:"checked_at"`
}

// Recount 根据 blocks/warnings 重算计数与 blocked。
func (r *ComplianceReport) Recount() {
	r.BlockCount = len(r.Blocks)
	r.WarningCount = len(r.Warnings)
	r.Blocked = r.BlockCount > 0
}

// LoadComplianceReport 读取 compliance-report.json。
func LoadComplianceReport(path string) (*ComplianceReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r ComplianceReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	if r.BlockCount == 0 && r.WarningCount == 0 && (len(r.Blocks) > 0 || len(r.Warnings) > 0) {
		r.Recount()
	}
	return &r, nil
}

// Save 写入 JSON 文件。
func (r *ComplianceReport) Save(path string) error {
	if r.CheckedAt == "" {
		r.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	}
	r.Recount()
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// Validate 基本字段校验。
func (r *ComplianceReport) Validate(episodeNo int) error {
	if r.EpisodeNo != 0 && r.EpisodeNo != episodeNo {
		return fmt.Errorf("episode_no mismatch: %d vs %d", r.EpisodeNo, episodeNo)
	}
	r.EpisodeNo = episodeNo
	r.Recount()
	return nil
}
