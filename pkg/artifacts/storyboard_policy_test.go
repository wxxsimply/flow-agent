package artifacts

import (
	"fmt"
	"testing"
)

func TestMicroMoviePolicySeedance(t *testing.T) {
	pol := MicroMoviePolicy("micro-movie-seedance")
	if pol.MinShots != 3 || pol.MaxShots != 8 {
		t.Fatalf("seedance policy: min=%d max=%d", pol.MinShots, pol.MaxShots)
	}
	if !pol.AllShotsAIVideo {
		t.Fatal("expected all shots ai video")
	}
}

func TestStoryboardValidateQuickStackEightShots(t *testing.T) {
	pol := MicroMoviePolicy("micro-movie-seedance")
	sb := &Storyboard{
		EpisodeNo:         1,
		TargetDurationSec: 45,
		Shots:             make([]Shot, 8),
	}
	for i := range sb.Shots {
		sb.Shots[i] = Shot{
			ID:            fmt.Sprintf("s%02d", i+1),
			DurationSec:   5,
			VisualType:    "ai_video",
			AIVideoBudget: true,
			VisualPrompt:  "test scene",
			Narration:     "旁白内容测试足够长一点",
			ActionBeats:   []string{"a", "b", "c"},
		}
	}
	if err := sb.Validate(1, 45, pol); err != nil {
		t.Fatalf("8 shots should pass seedance policy: %v", err)
	}
}
