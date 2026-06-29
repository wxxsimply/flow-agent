package config



import "strings"



// VoicePreset 旁白音色预设（前端可选）。

type VoicePreset struct {

	ID          string

	Label       string

	Description string

	VoiceID     string  // 传给 TTS provider 的 voice / voice_type

	SpeedRatio  float64 // <1 更慢，史诗感

}



var voicePresets = map[string]VoicePreset{

	"epic_male": {

		ID:          "epic_male",

		Label:       "史诗男声",

		Description: "低沉有力，叙事大片感",

		VoiceID:     "zh_male_m191_uranus_bigtts",

		SpeedRatio:  0.85,

	},

	"documentary_male": {

		ID:          "documentary_male",

		Label:       "纪录片男声",

		Description: "成熟稳重，纪实解说",

		VoiceID:     "zh_male_liufei_uranus_bigtts",

		SpeedRatio:  0.88,

	},

	"storyteller_male": {

		ID:          "storyteller_male",

		Label:       "说书男声",

		Description: "抑扬顿挫，传奇武侠",

		VoiceID:     "zh_male_liufei_uranus_bigtts",

		SpeedRatio:  0.86,

	},

	"warm_male": {

		ID:          "warm_male",

		Label:       "温暖男声",

		Description: "温和亲切，日常叙述",

		VoiceID:     "zh_male_m191_uranus_bigtts",

		SpeedRatio:  0.92,

	},

	"youthful_male": {

		ID:          "youthful_male",

		Label:       "青年男声",

		Description: "年轻活力，青春题材",

		VoiceID:     "zh_male_liufei_uranus_bigtts",

		SpeedRatio:  0.96,

	},

	"narrator_female": {

		ID:          "narrator_female",

		Label:       "女声旁白",

		Description: "清晰明亮，情感叙事",

		VoiceID:     "zh_female_vv_uranus_bigtts",

		SpeedRatio:  0.95,

	},

	"youthful_female": {

		ID:          "youthful_female",

		Label:       "青年女声",

		Description: "清新自然，都市青春",

		VoiceID:     "zh_female_xiaohe_uranus_bigtts",

		SpeedRatio:  0.94,

	},

	"calm_female": {

		ID:          "calm_female",

		Label:       "温柔女声",

		Description: "轻柔舒缓，治愈日常",

		VoiceID:     "zh_female_vv_uranus_bigtts",

		SpeedRatio:  0.93,

	},

	"dramatic_female": {

		ID:          "dramatic_female",

		Label:       "戏剧女声",

		Description: "张力十足，悬疑冲突",

		VoiceID:     "zh_female_xiaohe_uranus_bigtts",

		SpeedRatio:  0.90,

	},

	"bright_female": {

		ID:          "bright_female",

		Label:       "童声清新",

		Description: "明快童感，童话科普",

		VoiceID:     "zh_female_xiaohe_uranus_bigtts",

		SpeedRatio:  0.97,

	},

	"neutral_narrator": {

		ID:          "neutral_narrator",

		Label:       "中性旁白",

		Description: "客观平稳，资讯说明",

		VoiceID:     "zh_male_m191_uranus_bigtts",

		SpeedRatio:  0.90,

	},

}



// VoicePresetByID 返回音色预设；未知 id 原样透传为 VoiceID。

func VoicePresetByID(id string) VoicePreset {

	id = strings.TrimSpace(id)

	if p, ok := voicePresets[id]; ok {

		return p

	}

	if id == "" {

		return voicePresets["epic_male"]

	}

	return VoicePreset{ID: id, VoiceID: id, SpeedRatio: 0.85}

}



// ListVoicePresets 供 Web UI 展示。

func ListVoicePresets() []VoicePreset {

	order := []string{

		"epic_male", "documentary_male", "storyteller_male", "warm_male", "youthful_male", "neutral_narrator",

		"narrator_female", "youthful_female", "calm_female", "dramatic_female", "bright_female",

	}

	out := make([]VoicePreset, 0, len(order))

	for _, id := range order {

		out = append(out, voicePresets[id])

	}

	return out

}



// ResolveNarratorVoice 从创作参数解析 voice id 与语速。

func ResolveNarratorVoice(voiceID string, stackDefault string) (voice string, speed float64) {

	p := VoicePresetByID(voiceID)

	v := p.VoiceID

	if v == "" {

		v = stackDefault

	}

	if v == "" || v == "default" {

		v = "zh_male_m191_uranus_bigtts"

	}

	speed = p.SpeedRatio

	if speed <= 0 {

		speed = 0.85

	}

	return v, speed

}



// DashScopeVoiceFor 将火山/预设音色映射为百炼 Qwen/CosyVoice 可用音色（回退 TTS 时使用）。

func DashScopeVoiceFor(voiceID string) string {

	voiceID = strings.TrimSpace(voiceID)

	if voiceID == "" {

		return "longanyang"

	}

	if mapped, ok := dashScopeVoiceMap[voiceID]; ok {

		return mapped

	}

	switch voiceID {

	case "longanyang", "longwan", "longxiaochun", "longxiaobai":

		return voiceID

	}

	if strings.HasPrefix(voiceID, "BV") {

		if strings.Contains(strings.ToLower(voiceID), "female") ||

			strings.HasSuffix(voiceID, "001_streaming") ||

			strings.HasSuffix(voiceID, "007_streaming") {

			return "longwan"

		}

		return "longanyang"

	}

	return "longanyang"

}



var dashScopeVoiceMap = map[string]string{

	"BV406_streaming": "longanyang",

	"BV700_streaming": "longanyang",

	"BV002_streaming": "longanyang",

	"BV005_streaming": "longanyang",

	"BV001_streaming": "longwan",

	"BV007_streaming": "longwan",

	"zh_male_m191_uranus_bigtts":  "longanyang",
	"zh_male_liufei_uranus_bigtts": "longanyang",
	"zh_female_vv_uranus_bigtts":   "longwan",
	"zh_female_xiaohe_uranus_bigtts": "longwan",

	"longxiaochun":    "longxiaochun",

	"longxiaobai":     "longxiaobai",

}

