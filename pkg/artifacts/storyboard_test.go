package artifacts

import (
	"fmt"
	"strings"
	"testing"
)

func TestStoryboardValidate(t *testing.T) {
	sb := validTestStoryboard(180)
	if err := sb.Validate(1, 180); err != nil {
		t.Fatal(err)
	}
}

func TestStoryboardValidateKenBurnsShortDrama(t *testing.T) {
	sb := validKenBurnsStoryboard(90, 12)
	if err := sb.Validate(1, 90, KenBurnsShortDramaPolicy()); err != nil {
		t.Fatal(err)
	}
}

func TestStoryboardValidateVideoNative(t *testing.T) {
	sb := validVideoNativeStoryboard(90, 12)
	if err := sb.Validate(1, 90, VideoNativeShortPolicy()); err != nil {
		t.Fatal(err)
	}
	sb.Shots[0].VisualType = "ken_burns"
	sb.Shots[0].AIVideoBudget = false
	if err := sb.Validate(1, 90, VideoNativeShortPolicy()); err == nil {
		t.Fatal("expected error when ken_burns in video-native policy")
	}
}

func TestAlignDurationsFromNarration(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01", Narration: strings.Repeat("字", 40), DurationSec: 30},
			{ID: "s02", Narration: strings.Repeat("字", 20), DurationSec: 30},
		},
		TargetDurationSec: 120,
	}
	sb.AlignDurationsFromNarration(DefaultSecPerRune)
	total := sb.TotalDurationSec()
	if total < 14 || total > 16 {
		t.Fatalf("total=%.1f want ~15", total)
	}
	if sb.TargetDurationSec != total {
		t.Fatalf("target should match total, got %.1f vs %.1f", sb.TargetDurationSec, total)
	}
}

func validVideoNativeStoryboard(target, shotCount int) *Storyboard {
	targetF := float64(target)
	shotDur := targetF / float64(shotCount)
	shots := make([]Shot, shotCount)
	for i := 0; i < shotCount; i++ {
		shots[i] = Shot{
			ID:            fmt.Sprintf("s%02d", i+1),
			DurationSec:   shotDur,
			VisualType:    "ai_video",
			AIVideoBudget: true,
			VisualPrompt:  "竖屏9:16，跟拍，人物快步走过雨夜街道",
			Narration:     strings.Repeat("字", 8),
			Subtitle:      "字幕",
		}
	}
	sb := &Storyboard{
		EpisodeNo:         1,
		TargetDurationSec: targetF,
		Shots:             shots,
	}
	sb.NormalizeDurations(target)
	return sb
}

func TestStoryboardValidateRejectsBadAIVideoCount(t *testing.T) {
	sb := validTestStoryboard(180)
	sb.Shots[0].AIVideoBudget = false
	sb.Shots[0].VisualType = "ken_burns"
	if err := sb.Validate(1, 180); err == nil {
		t.Fatal("expected error for ai_video_budget count < 4")
	}
}

func TestStoryboardNormalizeDurations(t *testing.T) {
	sb := validTestStoryboard(180)
	for i := range sb.Shots {
		sb.Shots[i].DurationSec = 10
	}
	sb.NormalizeDurations(180)
	total := sb.TotalDurationSec()
	if total < 177 || total > 183 {
		t.Fatalf("total=%.1f want 177-183", total)
	}
}

func TestNarrationAlignmentScore(t *testing.T) {
	chapter := "雨夜，那通电话改变了一切。三年前的误会，如今真相浮现。他在天台等她。"
	sb := &Storyboard{
		Shots: []Shot{
			{Narration: "雨夜，那通电话改变了一切。"},
			{Narration: "三年前的误会，如今真相浮现。"},
			{Narration: "他在天台等她。"},
			{Narration: "完全不存在的旁白"},
		},
	}
	score, missing := NarrationAlignmentScore(chapter, sb)
	if score < 0.5 || score > 1 {
		t.Fatalf("score=%.2f", score)
	}
	if len(missing) == 0 {
		t.Fatal("expected missing narrations")
	}
}

func validTestStoryboard(target int) *Storyboard {
	targetF := float64(target)
	shotDur := targetF / 8
	shots := make([]Shot, 8)
	aiIdx := map[int]bool{0: true, 2: true, 5: true, 7: true}
	for i := 0; i < 8; i++ {
		ai := aiIdx[i]
		vt := "ken_burns"
		if ai {
			vt = "ai_video"
		}
		shots[i] = Shot{
			ID:            fmt.Sprintf("s%02d", i+1),
			DurationSec:   shotDur,
			VisualType:    vt,
			AIVideoBudget: ai,
			VisualPrompt:  "竖屏 9:16",
			Narration:     strings.Repeat("字", 10),
			Subtitle:      "字幕",
		}
	}
	sb := &Storyboard{
		EpisodeNo:         1,
		TargetDurationSec: targetF,
		Shots:             shots,
	}
	sb.NormalizeDurations(target)
	return sb
}

func validKenBurnsStoryboard(target, shotCount int) *Storyboard {
	targetF := float64(target)
	shotDur := targetF / float64(shotCount)
	shots := make([]Shot, shotCount)
	for i := 0; i < shotCount; i++ {
		shots[i] = Shot{
			ID:            fmt.Sprintf("s%02d", i+1),
			DurationSec:   shotDur,
			VisualType:    "ken_burns",
			AIVideoBudget: false,
			VisualPrompt:  "竖屏 9:16 短剧场景",
			Narration:     strings.Repeat("字", 8),
			Subtitle:      "字幕",
		}
	}
	sb := &Storyboard{
		EpisodeNo:         1,
		TargetDurationSec: targetF,
		Shots:             shots,
	}
	sb.NormalizeDurations(target)
	return sb
}
