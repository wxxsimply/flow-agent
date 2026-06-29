package provider

import (
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/provider/image"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/provider/tts"
	"github.com/flow-agent/flow-agent/internal/provider/video"
)

// Bundle 一次运行所需的 Provider 客户端集合。
type Bundle struct {
	DeepSeek llm.Client
	Bailian  llm.Client
	TTS      tts.Client
	Image    image.Client
	Video    video.Client
}

// NewBundle 根据 App 配置创建客户端；无 Key 时使用 Noop。
func NewBundle(app *config.App) *Bundle {
	b := &Bundle{
		DeepSeek: llm.Noop{},
		Bailian:  llm.Noop{},
		TTS:      tts.Noop{},
		Image:    image.Noop{},
		Video:    video.Noop{},
	}
	if app.Providers.DeepSeek.APIKey != "" {
		b.DeepSeek = llm.NewDeepSeek(app.Providers.DeepSeek)
	}
	if strings.TrimSpace(app.Providers.DashScope.APIKey) != "" {
		b.Bailian = llm.NewBailian(app.Providers)
	}

	imgProv := ""
	vidProv := ""
	if app.Stack != nil {
		imgProv = strings.ToLower(strings.TrimSpace(app.Stack.ImageConfig().Provider))
		vidProv = strings.ToLower(strings.TrimSpace(app.Stack.VideoConfig().Provider))
	}

	switch imgProv {
	case "volcengine", "seedream", "jimeng", "ark":
		if app.Providers.VolcengineArkConfigured() {
			model := ""
			if app.Stack != nil {
				model = app.Stack.ImageConfig().Model
			}
			b.Image = image.NewVolcengine(app.Providers, model)
		}
	case "openai", "sora":
		if app.Providers.OpenAIEnabled() {
			model := ""
			if app.Stack != nil {
				model = app.Stack.ImageConfig().Model
			}
			b.Image = image.NewOpenAIMedia(
				app.Providers.OpenAI.APIKey,
				app.Providers.OpenAI.BaseURL,
				model,
			)
		}
	case "gemini", "veo", "google":
		if app.Providers.GeminiEnabled() {
			model := ""
			if app.Stack != nil {
				model = app.Stack.ImageConfig().Model
			}
			b.Image = image.NewOpenAIMedia(
				app.Providers.Gemini.APIKey,
				app.Providers.Gemini.BaseURL,
				model,
			)
		}
	default:
		if strings.TrimSpace(app.Providers.DashScope.APIKey) != "" {
			imgModel := ""
			if app.Stack != nil {
				imgModel = app.Stack.ImageConfig().Model
			}
			b.Image = image.NewDashScope(app.Providers, imgModel)
		}
	}

	b.TTS = selectTTS(app)

	videoOn := false
	if app.Stack != nil {
		videoOn = app.Stack.VideoConfig().Enabled
	}
	if videoOn {
		vc := app.Stack.VideoConfig()
		switch vidProv {
		case "volcengine", "seedance", "seedream", "jimeng", "ark":
			if app.Providers.VolcengineArkConfigured() {
				b.Video = video.NewVolcengine(app.Providers, vc.Model, vc.AspectRatio, vc.SilentAudio, vc.FastPoll)
			}
		case "gemini", "veo", "google":
			if app.Providers.GeminiEnabled() {
				b.Video = video.NewGeminiVeo(app.Providers, vc.Model, vc.AspectRatio, vc.Resolution, vc.FastPoll)
			}
		case "openai", "sora":
			if app.Providers.OpenAIEnabled() {
				b.Video = video.NewSora(app.Providers, vc.Model, vc.AspectRatio, vc.Resolution, vc.FastPoll)
			}
		case "bailian", "dashscope", "wan":
			if strings.TrimSpace(app.Providers.DashScope.APIKey) != "" {
				b.Video = video.NewWan(app.Providers, vc.Model, vc.QualityModel, vc.Resolution, vc.SilentAudio, vc.FastPoll)
			}
		case "kling":
			if app.Providers.KlingEnabled() {
				b.Video = video.NewKling(app.Providers, vc.Image2VideoModel, vc.TextModel)
			}
		}
	}
	return b
}

func selectTTS(app *config.App) tts.Client {
	p := app.Providers
	dashKey := strings.TrimSpace(p.DashScope.APIKey)
	ttsProv := ""
	if app.Stack != nil {
		ttsProv = strings.ToLower(strings.TrimSpace(app.Stack.TTSConfig().Provider))
	}

	var vol tts.Client
	if p.VolcengineTTSConfigured() {
		vol = tts.NewVolcengine(p)
	}
	var dash tts.Client
	if dashKey != "" {
		dash = tts.NewDashScope(p)
	}

	forceVolc := os.Getenv("FLOWAGENT_TTS_PROVIDER") == "volcengine"
	allowDashFallback := os.Getenv("FLOWAGENT_TTS_FALLBACK") == "dashscope"

	switch ttsProv {
	case "dashscope", "bailian", "qwen":
		if dash != nil {
			return dash
		}
	case "volcengine":
		if vol != nil {
			if allowDashFallback && dash != nil {
				return tts.NewResourceGrantFallback(vol, dash)
			}
			return vol
		}
		return tts.Noop{}
	}

	if forceVolc {
		if vol != nil {
			return vol
		}
		if allowDashFallback && dash != nil {
			return dash
		}
		return tts.Noop{}
	}

	if allowDashFallback && vol != nil && dash != nil {
		return tts.NewFallback(vol, dash)
	}
	if vol != nil {
		return vol
	}
	if dash != nil {
		return dash
	}
	return tts.Noop{}
}

// ClientFor 按 stack provider 名选择 LLM 客户端。
func (b *Bundle) ClientFor(providerName string) llm.Client {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "bailian", "dashscope", "qwen":
		if b.Bailian != nil {
			return b.Bailian
		}
	case "deepseek":
		if b.DeepSeek != nil {
			return b.DeepSeek
		}
	}
	if b.DeepSeek != nil {
		return b.DeepSeek
	}
	return llm.Noop{}
}

// LLMForStage 读取 stack 中某阶段的 provider 并返回对应客户端。
func (b *Bundle) LLMForStage(app *config.App, stage string) llm.Client {
	if b == nil {
		return llm.Noop{}
	}
	ref := app.LLMRef(stage)
	return b.ClientFor(ref.Provider)
}
