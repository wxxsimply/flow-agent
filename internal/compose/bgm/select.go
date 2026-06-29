package bgm

import (
	"os"
	"path/filepath"
	"strings"
)

// Mood 曲库情绪标签。
type Mood string

const (
	MoodNeutral  Mood = "neutral"
	MoodTense    Mood = "tense"
	MoodSuspense Mood = "suspense"
	MoodSad      Mood = "sad"
	MoodWarm     Mood = "warm"
	MoodHopeful  Mood = "hopeful"
	MoodEpic     Mood = "epic"
	MoodRomantic Mood = "romantic"
	MoodHorror   Mood = "horror"
	MoodComedy   Mood = "comedy"
)

// SelectPath 根据情绪词在曲库目录中选 BGM 文件。
func SelectPath(libraryDir, mood, tone string) (path string, resolved Mood, ok bool) {
	libraryDir = strings.TrimSpace(libraryDir)
	if libraryDir == "" {
		return "", "", false
	}
	key := NormalizeMood(mood, tone)
	candidates := []Mood{key, MoodNeutral}
	// 相邻情绪回退
	switch key {
	case MoodSuspense:
		candidates = append([]Mood{MoodTense}, candidates...)
	case MoodTense:
		candidates = append([]Mood{MoodSuspense}, candidates...)
	case MoodHopeful:
		candidates = append([]Mood{MoodWarm}, candidates...)
	}
	seen := map[Mood]bool{}
	for _, m := range candidates {
		if seen[m] {
			continue
		}
		seen[m] = true
		p := filepath.Join(libraryDir, string(m)+".mp3")
		if fileExists(p) {
			return p, m, true
		}
	}
	return "", key, false
}

// NormalizeMood 将 LLM 输出的 tone/mood 映射到曲库文件名。
func NormalizeMood(mood, tone string) Mood {
	s := strings.ToLower(strings.TrimSpace(mood + " " + tone))
	switch {
	case strings.Contains(s, "悬疑") || strings.Contains(s, "紧张") || strings.Contains(s, "惊悚"):
		return MoodSuspense
	case strings.Contains(s, "恐怖") || strings.Contains(s, "阴森"):
		return MoodHorror
	case strings.Contains(s, "悲") || strings.Contains(s, "哀") || strings.Contains(s, "伤感"):
		return MoodSad
	case strings.Contains(s, "温") || strings.Contains(s, "治愈") || strings.Contains(s, "暖"):
		return MoodWarm
	case strings.Contains(s, "希望") || strings.Contains(s, "励志") || strings.Contains(s, "昂扬"):
		return MoodHopeful
	case strings.Contains(s, "史诗") || strings.Contains(s, "宏大") || strings.Contains(s, "燃"):
		return MoodEpic
	case strings.Contains(s, "浪漫") || strings.Contains(s, "爱情"):
		return MoodRomantic
	case strings.Contains(s, "喜剧") || strings.Contains(s, "轻松") || strings.Contains(s, "搞笑"):
		return MoodComedy
	case strings.Contains(s, "tense") || strings.Contains(s, "suspense"):
		return MoodSuspense
	case strings.Contains(s, "sad") || strings.Contains(s, "melanch"):
		return MoodSad
	case strings.Contains(s, "warm") || strings.Contains(s, "cozy"):
		return MoodWarm
	case strings.Contains(s, "epic") || strings.Contains(s, "heroic"):
		return MoodEpic
	case strings.Contains(s, "horror"):
		return MoodHorror
	case strings.Contains(s, "romantic"):
		return MoodRomantic
	case strings.Contains(s, "comedy") || strings.Contains(s, "fun"):
		return MoodComedy
	default:
		return MoodNeutral
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
