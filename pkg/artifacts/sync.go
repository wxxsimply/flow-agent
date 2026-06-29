package artifacts

import (
	"encoding/json"
	"os"
)

// AudioSegment 单镜旁白在时间轴上的位置（K1）。
type AudioSegment struct {
	ShotID      string  `json:"shot_id"`
	Path        string  `json:"path"`
	StartSec    float64 `json:"start_sec"`
	DurationSec float64 `json:"duration_sec"`
}

// AudioSegments 按镜 TTS 分段清单。
type AudioSegments struct {
	Segments  []AudioSegment `json:"segments"`
	TotalSec  float64        `json:"total_sec"`
	Narration string         `json:"narration_path"`
}

// Save 写入 audio_segments.json。
func (a *AudioSegments) Save(path string) error {
	b, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// SyncReport 音画对齐报告（K1 / 门禁 av_sync_ok）。
type SyncReport struct {
	VideoTotalSec   float64 `json:"video_total_sec"`
	AudioTotalSec   float64 `json:"audio_total_sec"`
	SpeechAudioSec  float64 `json:"speech_audio_sec,omitempty"` // TTS 实测旁白，不含尾部 Pad 静音
	MaxDriftSec     float64 `json:"max_drift_sec"`
	Aligned         bool    `json:"aligned"`
	ShotAudioPadSec float64 `json:"shot_audio_pad_sec"`
}

// Save 写入 sync-report.json。
func (r *SyncReport) Save(path string) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadAudioSegments 读取 audio_segments.json。
func LoadAudioSegments(path string) (*AudioSegments, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a AudioSegments
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// LoadSyncReport 读取 sync-report.json。
func LoadSyncReport(path string) (*SyncReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r SyncReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
