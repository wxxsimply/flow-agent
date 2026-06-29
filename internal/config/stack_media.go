package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// StackImageConfig 从 stack 读取出图配置。
type StackImageConfig struct {
	Provider          string
	Model             string
	AspectRatio       string
	Width             int
	Height            int
	StylePrefix       string
	UseTurnaroundSeed bool
}

// StackTTSConfig TTS 配置。
type StackTTSConfig struct {
	Provider   string
	Voice      string
	Format     string
	Product    string // 如 doubao-speech-2.0-emotion
	ResourceID string // 可选，显式 seed-tts-2.0 / seed-tts-1.0
}

// StackVideoConfig AI 视频生成配置（万相 / 可灵等）。
type StackVideoConfig struct {
	Enabled              bool
	Provider             string
	Model                string
	QualityModel         string // 万相：画质回退模型（如 wan2.6-i2v）
	Resolution           string // 万相：720P / 1080P
	SilentAudio          bool   // 万相 flash：audio=false 生成无声视频（旁白由 TTS 轨）
	Mode                 string
	MaxClips             int
	Strategy             string // text2video | image2video
	AllShots             bool
	RequireVideo         bool
	SkipImage            bool
	FallbackImage2Video  bool
	AspectRatio          string
	TextModel            string // 文生视频 model_name（可灵 text2video）
	Image2VideoModel     string // 图生视频 model_name（可灵 image2video）
	MotionPromptSuffix   string // 追加到每镜图生视频 prompt，强化动效
	ClipDurationSec      int    // 分镜默认单镜时长（秒）；不作为 i2v API 硬顶（i2v 对齐 TTS 实测）
	WMRewardBoNEnabled    bool   // 每镜多候选 i2v，物理选优（启发式或 WMReward 脚本）
	WMRewardBoNCandidates int
	WMRewardBoNHeroOnly   bool   // tiered 时仅 hero 镜启用 BoN
	WMRewardScriptPath    string // 可选：python script，stdout 输出 surprise 浮点（越低越好）
	KeyframeMode          string // single | multi | tiered
	InterShotDelaySec     int    // 串行 produce 时镜间等待（并行时忽略）
	MaxParallelShots      int    // produce 并行度，默认 1
	MaxProduceShots       int    // produce 前截断镜数上限，0 不限制
	FastPoll              bool   // 万相轮询间隔缩短（约 8s）
	HeroShotCount         int    // tiered 时标为 hero 的镜数（含 s01 与末镜）
	UseQualityModelFor    string // none | hero | always
	HeroResolution        string // hero 镜 i2v 分辨率，空则用 Resolution
	HeroBonCandidates     int    // hero 镜 BoN 候选数，0 则用 WMRewardBoNCandidates
	ChainShotKeyframes    bool   // 兼容旧配置；true 等价 chain_shot_mode=hard
	ChainShotMode         string // off | soft | hard：镜间衔接强度
	KenBurnsFallback      bool   // i2v 失败时用关键帧 Ken Burns 生成 mp4
	EnforceClipDurationCap bool  // 低成本栈：i2v 与预算按 clip_duration_sec 封顶
}

// ChainMode 返回镜间衔接模式。
func (c StackVideoConfig) ChainMode() string {
	switch strings.ToLower(strings.TrimSpace(c.ChainShotMode)) {
	case "off", "none", "false":
		return "off"
	case "hard", "strict", "frame":
		return "hard"
	case "soft", "relaxed", "prompt":
		return "soft"
	}
	if c.ChainShotKeyframes {
		return "hard"
	}
	return "off"
}

// StackAssembleConfig assemble 阶段开关。
type StackAssembleConfig struct {
	LLMReview           bool
	QuickAssemble       bool
	CharacterTurnaround bool
	PropTurnaround      bool
	BriefRunesMin       int
	BriefRunesMax       int
	BriefRunesFloor     int
	BriefSegmentCount   int
}

const (
	defaultBriefRunesMin     = 2800
	defaultBriefRunesMax     = 3200
	defaultBriefRunesFloor   = 2200
	defaultBriefSegmentCount = 3
)

// AssembleConfig 解析 stack.assemble。
func (s *Stack) AssembleConfig() StackAssembleConfig {
	cfg := StackAssembleConfig{
		LLMReview:         true,
		BriefRunesMin:     defaultBriefRunesMin,
		BriefRunesMax:     defaultBriefRunesMax,
		BriefRunesFloor:   defaultBriefRunesFloor,
		BriefSegmentCount: defaultBriefSegmentCount,
	}
	if s == nil || s.Assemble == nil {
		return cfg
	}
	if v, ok := s.Assemble["llm_review"].(bool); ok {
		cfg.LLMReview = v
	}
	if v, ok := s.Assemble["quick_assemble"].(bool); ok {
		cfg.QuickAssemble = v
	}
	if v, ok := s.Assemble["character_turnaround"].(bool); ok {
		cfg.CharacterTurnaround = v
	}
	if v, ok := s.Assemble["prop_turnaround"].(bool); ok {
		cfg.PropTurnaround = v
	}
	if v := intFromMap(s.Assemble, "brief_runes_min"); v > 0 {
		cfg.BriefRunesMin = v
	}
	if v := intFromMap(s.Assemble, "brief_runes_max"); v > 0 {
		cfg.BriefRunesMax = v
	}
	if v := intFromMap(s.Assemble, "brief_runes_floor"); v > 0 {
		cfg.BriefRunesFloor = v
	}
	if v := intFromMap(s.Assemble, "brief_segment_count"); v > 0 {
		cfg.BriefSegmentCount = v
	}
	return cfg
}

func intFromMap(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// VideoNative 方案 B：全镜 AI 视频，成片不依赖 Ken Burns。
func (c StackVideoConfig) VideoNative() bool {
	return c.Enabled && c.AllShots
}

// ImageConfig 解析 stack.image。
func (s *Stack) ImageConfig() StackImageConfig {
	cfg := StackImageConfig{
		Provider:    strMap(s.Image, "provider", "bailian"),
		Model:       strMap(s.Image, "model", "wan2.6-t2i"),
		AspectRatio: strMap(s.Image, "aspect_ratio", "9:16"),
		Width:       1080,
		Height:      1920,
	}
	if ar, ok := s.Image["resolution"].(string); ok && ar != "" {
		if w, h, ok := parseResolution(ar); ok {
			cfg.Width, cfg.Height = w, h
		}
	}
	if sp, ok := s.Image["style_prefix"].(string); ok {
		cfg.StylePrefix = sp
	}
	if v, ok := s.Image["use_turnaround_seed"].(bool); ok {
		cfg.UseTurnaroundSeed = v
	}
	return cfg
}

// TTSConfig 解析 stack.tts。
func (s *Stack) TTSConfig() StackTTSConfig {
	return StackTTSConfig{
		Provider:   strMap(s.TTS, "provider", "volcengine"),
		Voice:      strMap(s.TTS, "voices", "narrator"),
		Format:     strMap(s.TTS, "format", "mp3"),
		Product:    strMap(s.TTS, "product", "doubao-speech-2.0-emotion"),
		ResourceID: strMap(s.TTS, "resource_id", ""),
	}
}

// StackComposeConfig FFmpeg 合成（阶段 K）。
type StackComposeConfig struct {
	SubtitleFormat       string
	ShotAudioPadSec      float64
	BGMEnabled           bool
	BGMPath              string
	BGMLibrary           string
	BGMVolume            float64
	ClipCrossfadeSec     float64
	ClipEdgeFadeSec      float64 // 单镜首尾淡入淡出，减轻硬切
	SubtitleCharsPerLine int
	SubtitleMaxLines     int
	VideoNativeOnly      bool // 仅使用 mp4 镜头，禁止 Ken Burns
}

// ComposeConfig 解析 stack.compose。
func (s *Stack) ComposeConfig() StackComposeConfig {
	cfg := StackComposeConfig{
		SubtitleFormat:       "ass",
		ShotAudioPadSec:      0.15,
		BGMVolume:            0.25,
		ClipCrossfadeSec:     0.35,
		ClipEdgeFadeSec:      0.12,
		SubtitleCharsPerLine: 14,
		SubtitleMaxLines:     3,
	}
	if s == nil || s.Compose == nil {
		return cfg
	}
	if v, ok := s.Compose["subtitle_format"].(string); ok && v != "" {
		cfg.SubtitleFormat = v
	}
	if v, ok := s.Compose["shot_audio_pad_sec"].(float64); ok && v > 0 {
		cfg.ShotAudioPadSec = v
	}
	if v, ok := s.Compose["bgm_enabled"].(bool); ok {
		cfg.BGMEnabled = v
	}
	if v, ok := s.Compose["bgm_path"].(string); ok {
		cfg.BGMPath = v
	}
	if v, ok := s.Compose["bgm_library"].(string); ok {
		cfg.BGMLibrary = v
	}
	if v, ok := s.Compose["bgm_volume"].(float64); ok && v > 0 {
		cfg.BGMVolume = v
	}
	if v, ok := s.Compose["clip_crossfade_sec"].(float64); ok {
		cfg.ClipCrossfadeSec = v
	}
	if v, ok := s.Compose["clip_edge_fade_sec"].(float64); ok && v >= 0 {
		cfg.ClipEdgeFadeSec = v
	}
	if v, ok := s.Compose["subtitle_chars_per_line"].(int); ok && v > 0 {
		cfg.SubtitleCharsPerLine = v
	}
	if v, ok := s.Compose["subtitle_max_lines"].(int); ok && v > 0 {
		cfg.SubtitleMaxLines = v
	}
	if v, ok := s.Compose["video_native_only"].(bool); ok {
		cfg.VideoNativeOnly = v
	}
	return cfg
}

// VideoConfig 解析 stack.video。
func (s *Stack) VideoConfig() StackVideoConfig {
	cfg := StackVideoConfig{
		Enabled:             false,
		Provider:            "kling",
		Model:               "kling-v2-5-turbo",
		Mode:                "std",
		MaxClips:            6,
		Strategy:            "text2video",
		AspectRatio:         "9:16",
		RequireVideo:        true,
		FallbackImage2Video: true,
	}
	if s == nil || s.Video == nil {
		return cfg
	}
	cfg.Provider = strMap(s.Video, "provider", cfg.Provider)
	cfg.Model = strMap(s.Video, "model", cfg.Model)
	cfg.TextModel = strMap(s.Video, "text_model", cfg.TextModel)
	cfg.Image2VideoModel = strMap(s.Video, "image2video_model", cfg.Image2VideoModel)
	cfg.QualityModel = strMap(s.Video, "quality_model", cfg.QualityModel)
	cfg.Resolution = strMap(s.Video, "resolution", cfg.Resolution)
	cfg.MotionPromptSuffix = strMap(s.Video, "motion_prompt_suffix", cfg.MotionPromptSuffix)
	if v, ok := s.Video["silent_audio"].(bool); ok {
		cfg.SilentAudio = v
	}
	if v, ok := s.Video["clip_duration_sec"].(int); ok && v > 0 {
		cfg.ClipDurationSec = v
	}
	if v, ok := s.Video["clip_duration_sec"].(float64); ok && v > 0 {
		cfg.ClipDurationSec = int(v)
	}
	cfg.Mode = strMap(s.Video, "mode", cfg.Mode)
	cfg.Strategy = strMap(s.Video, "strategy", cfg.Strategy)
	cfg.AspectRatio = strMap(s.Video, "aspect_ratio", cfg.AspectRatio)
	if v, ok := s.Video["enabled"].(bool); ok {
		cfg.Enabled = v
	}
	if v, ok := s.Video["all_shots"].(bool); ok {
		cfg.AllShots = v
	}
	if v, ok := s.Video["skip_image"].(bool); ok {
		cfg.SkipImage = v
	}
	if v, ok := s.Video["require_video"].(bool); ok {
		cfg.RequireVideo = v
	}
	if v, ok := s.Video["fallback_image2video"].(bool); ok {
		cfg.FallbackImage2Video = v
	}
	if v, ok := s.Video["max_clips"].(int); ok {
		cfg.MaxClips = v
	}
	cfg.KeyframeMode = strMap(s.Video, "keyframe_mode", "multi")
	if v, ok := s.Video["inter_shot_delay_sec"].(int); ok && v >= 0 {
		cfg.InterShotDelaySec = v
	}
	if v, ok := s.Video["inter_shot_delay_sec"].(float64); ok && v >= 0 {
		cfg.InterShotDelaySec = int(v)
	}
	if cfg.InterShotDelaySec == 0 && s.Name == "micro-movie-wan-flash" {
		cfg.InterShotDelaySec = 1 // 默认 1s，替代原硬编码 8s
	}
	if v, ok := s.Video["max_parallel_shots"].(int); ok && v > 0 {
		cfg.MaxParallelShots = v
	}
	if v, ok := s.Video["max_parallel_shots"].(float64); ok && v > 0 {
		cfg.MaxParallelShots = int(v)
	}
	if v, ok := s.Video["max_produce_shots"].(int); ok && v > 0 {
		cfg.MaxProduceShots = v
	}
	if v, ok := s.Video["max_produce_shots"].(float64); ok && v > 0 {
		cfg.MaxProduceShots = int(v)
	}
	if v, ok := s.Video["fast_poll"].(bool); ok {
		cfg.FastPoll = v
	}
	if v, ok := s.Video["hero_shot_count"].(int); ok && v > 0 {
		cfg.HeroShotCount = v
	}
	if v, ok := s.Video["hero_shot_count"].(float64); ok && v > 0 {
		cfg.HeroShotCount = int(v)
	}
	if cfg.HeroShotCount <= 0 {
		cfg.HeroShotCount = 3
	}
	cfg.UseQualityModelFor = strMap(s.Video, "use_quality_model_for", "none")
	if hero, ok := s.Video["hero"].(map[string]any); ok {
		if v, ok := hero["resolution"].(string); ok {
			cfg.HeroResolution = v
		}
		if v, ok := hero["bon_candidates"].(int); ok && v > 0 {
			cfg.HeroBonCandidates = v
		}
		if v, ok := hero["bon_candidates"].(float64); ok && v > 0 {
			cfg.HeroBonCandidates = int(v)
		}
	}
	if wm, ok := s.Video["wmreward_bon"].(map[string]any); ok {
		if v, ok := wm["enabled"].(bool); ok {
			cfg.WMRewardBoNEnabled = v
		}
		if v, ok := wm["hero_only"].(bool); ok {
			cfg.WMRewardBoNHeroOnly = v
		}
		if v, ok := wm["candidates"].(int); ok && v > 0 {
			cfg.WMRewardBoNCandidates = v
		}
		if v, ok := wm["candidates"].(float64); ok && v > 0 {
			cfg.WMRewardBoNCandidates = int(v)
		}
		if v, ok := wm["script_path"].(string); ok {
			cfg.WMRewardScriptPath = v
		}
	}
	if cfg.WMRewardBoNCandidates <= 0 {
		cfg.WMRewardBoNCandidates = 3
	}
	if cfg.AllShots {
		cfg.MaxClips = 0
		if !cfg.SkipImage && cfg.Strategy == "text2video" {
			cfg.SkipImage = true
		}
	}
	if v, ok := s.Video["chain_shot_keyframes"].(bool); ok {
		cfg.ChainShotKeyframes = v
	}
	if v, ok := s.Video["chain_shot_mode"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.ChainShotMode = strings.TrimSpace(v)
	} else if s.Name == "micro-movie-seedance" {
		cfg.ChainShotMode = "soft"
	} else if s.Name == "micro-movie-wan-flash" || s.Name == "micro-movie-wan-fast" ||
		s.Name == "micro-movie-wan-quick" || s.Name == "micro-movie-wan-premiere" ||
		s.Name == "micro-movie-economy" {
		cfg.ChainShotKeyframes = true
	}
	if v, ok := s.Video["ken_burns_fallback"].(bool); ok {
		cfg.KenBurnsFallback = v
	} else if s.Name == "micro-movie-seedance" {
		cfg.KenBurnsFallback = true
	}
	if v, ok := s.Video["enforce_clip_duration_cap"].(bool); ok {
		cfg.EnforceClipDurationCap = v
	}
	if !cfg.EnforceClipDurationCap && s.CostBudgetCNY > 0 && s.CostBudgetCNY <= 5 {
		cfg.EnforceClipDurationCap = true
	}
	if !cfg.EnforceClipDurationCap && s.CostBudgetPer30SecCNY > 0 {
		cfg.EnforceClipDurationCap = true
	}
	return cfg
}

func parseResolution(s string) (w, h int, ok bool) {
	parts := strings.Split(strings.TrimSpace(s), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return 0, 0, false
	}
	h, err = strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

// StoryboardPolicy 按 stack 视频档位返回分镜策略。
func (s *Stack) StoryboardPolicy() artifacts.StoryboardPolicy {
	if s == nil {
		return artifacts.KenBurnsShortDramaPolicy()
	}
	vid := s.VideoConfig()
	if s.Name == "micro-movie-wan-flash" || s.Name == "micro-movie-wan-hd" ||
		s.Name == "micro-movie-economy" || s.Name == "micro-movie-wan-fast" ||
		s.Name == "micro-movie-wan-quick" || s.Name == "micro-movie-wan-premiere" ||
		s.Name == "micro-movie-seedance" || s.Name == "micro-movie-cap5" {
		return artifacts.MicroMoviePolicy(s.Name)
	}
	if vid.VideoNative() {
		return artifacts.VideoNativeShortPolicy()
	}
	if vid.Enabled {
		return artifacts.DefaultAIVideoPolicy()
	}
	return artifacts.KenBurnsShortDramaPolicy()
}

func strMap(m map[string]any, key, def string) string {
	if m == nil {
		return def
	}
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	if key == "voices" {
		if voices, ok := m["voices"].(map[string]any); ok {
			if n, ok := voices["narrator"].(string); ok && n != "" {
				return n
			}
		}
	}
	return def
}

// ImageSizeParam 返回万相 API size 参数字符串。
func (c StackImageConfig) ImageSizeParam() string {
	return fmt.Sprintf("%d*%d", c.Width, c.Height)
}
