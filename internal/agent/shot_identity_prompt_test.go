package agent

import (
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestShotIdentityNegSuffix(t *testing.T) {
	neg := shotIdentityNegSuffix(artifacts.Shot{
		CharacterCount: 1,
		HeldProps:      artifacts.FlexString("右手：匕首；左手：空"),
	})
	if neg == "" || !strings.Contains(neg, "extra weapons") {
		t.Fatalf("got %q", neg)
	}
	if !strings.Contains(neg, "道具换手") {
		t.Fatalf("expected prop hand neg, got %q", neg)
	}
}

func TestShotPropLockSuffix(t *testing.T) {
	s := shotPropLockSuffix(artifacts.Shot{HeldProps: artifacts.FlexString("右手：匕首；左手：空")})
	if !strings.Contains(s, "[PROP_LOCK]") {
		t.Fatalf("got %q", s)
	}
}

func TestKeyframeBeatForShot_prefersMatchingBeat(t *testing.T) {
	shot := artifacts.Shot{
		HeldProps:    artifacts.FlexString("右手：匕首；左手：空"),
		VisualPrompt: "右手横握匕首，中景",
		ActionBeats: []string{
			"少年迈步，左手摆动",
			"右手横握匕首，刀尖向右",
			"收势停步",
		},
	}
	beat := keyframeBeatForShot(shot, false)
	if !strings.Contains(beat, "横握") {
		t.Fatalf("expected matching beat, got %q", beat)
	}
}

func TestKeyframeBeatForShot_lastBeatFallback(t *testing.T) {
	shot := artifacts.Shot{
		HeldProps:    artifacts.FlexString("右手：匕首；左手：空"),
		VisualPrompt: "远景站立",
		ActionBeats: []string{
			"迈步",
			"停步",
			"右手持匕首站立",
		},
	}
	beat := keyframeBeatForShot(shot, false)
	if beat != "右手持匕首站立" {
		t.Fatalf("got %q", beat)
	}
}
