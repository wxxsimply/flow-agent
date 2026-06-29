package artifacts

import (
	"encoding/json"
	"os"
)

// Timeline 由 storyboard 驱动的合成时间轴（produce 阶段产物，含 v2 音画字段）。
type Timeline struct {
	EpisodeNo    int             `json:"episode_no"`
	FPS          int             `json:"fps"`
	Resolution   string          `json:"resolution"`
	Tracks       []TimelineTrack `json:"tracks"`
	Shots        []TimelineShot  `json:"shots"`
	Audio        *TimelineAudio  `json:"audio,omitempty"`
	SubtitleFile string          `json:"subtitle_file,omitempty"`
}

// TimelineTrack 顶层轨道描述。
type TimelineTrack struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Optional bool   `json:"optional,omitempty"`
}

// TimelineAudio 顶层音频描述（timeline v2）。
type TimelineAudio struct {
	Narration string  `json:"narration"`
	BGM       string  `json:"bgm,omitempty"`
	TotalSec  float64 `json:"total_sec,omitempty"`
}

// TimelineShot 单镜头合成指令。
type TimelineShot struct {
	ID                string  `json:"id"`
	DurationSec       float64 `json:"duration_sec"`
	AudioStartSec     float64 `json:"audio_start_sec,omitempty"`
	AudioDurationSec  float64 `json:"audio_duration_sec,omitempty"`
	VideoDurationSec  float64 `json:"video_duration_sec,omitempty"` // produce 后 ffprobe 实测
	VisualType        string  `json:"visual_type"`
	AIVideoBudget     bool    `json:"ai_video_budget,omitempty"`
	ImagePath         string  `json:"image_path,omitempty"`
	VideoPath         string  `json:"video_path,omitempty"`
	Narration         string  `json:"narration"`
	Subtitle          string  `json:"subtitle"`
}

// BuildTimeline 从 storyboard 构建 timeline.json 结构。
func BuildTimeline(sb *Storyboard, episodeNo int) *Timeline {
	tl := &Timeline{
		EpisodeNo:  episodeNo,
		FPS:        30,
		Resolution: "1080x1920",
		Tracks: []TimelineTrack{
			{Type: "video", Source: ShotsDir + "/"},
			{Type: "audio", Source: MediaDir + "/narration.mp3"},
		},
	}
	for _, s := range sb.Shots {
		ts := TimelineShot{
			ID:            s.ID,
			DurationSec:   s.DurationSec,
			VisualType:    s.VisualType,
			AIVideoBudget: s.AIVideoBudget,
			ImagePath:     ShotImageRel(s.ID),
			VideoPath:     ShotVideoRel(s.ID),
			Narration:     s.Narration,
			Subtitle:      s.Subtitle,
		}
		tl.Shots = append(tl.Shots, ts)
	}
	return tl
}

// BuildTimelineAligned 以实测音频时长构建 timeline（K1）。
func BuildTimelineAligned(sb *Storyboard, episodeNo int, segments []AudioSegment, padSec float64) *Timeline {
	tl := BuildTimeline(sb, episodeNo)
	tl.Audio = &TimelineAudio{
		Narration: MediaDir + "/narration.mp3",
		TotalSec:  0,
	}
	var videoTotal float64
	for i := range tl.Shots {
		if i >= len(segments) {
			break
		}
		seg := segments[i]
		audioDur := seg.DurationSec
		if audioDur <= 0 {
			audioDur = tl.Shots[i].DurationSec
		}
		tl.Shots[i].AudioStartSec = seg.StartSec
		tl.Shots[i].AudioDurationSec = audioDur
		tl.Shots[i].DurationSec = audioDur + padSec
		videoTotal += tl.Shots[i].DurationSec
	}
	if tl.Audio != nil {
		if len(segments) > 0 {
			last := segments[len(segments)-1]
			tl.Audio.TotalSec = last.StartSec + last.DurationSec
		} else {
			tl.Audio.TotalSec = videoTotal
		}
	}
	return tl
}

// TotalVideoSec 各镜视频时长之和。
func (t *Timeline) TotalVideoSec() float64 {
	var sum float64
	for _, s := range t.Shots {
		sum += s.DurationSec
	}
	return sum
}

// Save 写入 timeline.json。
func (t *Timeline) Save(path string) error {
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// LoadTimeline 读取 timeline.json。
func LoadTimeline(path string) (*Timeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Timeline
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
