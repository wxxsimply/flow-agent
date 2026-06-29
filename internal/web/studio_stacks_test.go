package web

import (
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestDefaultStudioStack_isSample(t *testing.T) {
	if DefaultStudioStack != StudioStackSample {
		t.Fatalf("want %q, got %q", StudioStackSample, DefaultStudioStack)
	}
}

func TestNormalizeStudioStack(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", StudioStackSample},
		{StudioStackSample, StudioStackSample},
		{StudioStackFinal, StudioStackFinal},
		{"micro-movie-wan-flash", StudioStackFinal},
		{"micro-movie-sora", StudioStackFinal},
		{"micro-movie-economy", StudioStackSample},
		{"micro-movie-wan-quick", StudioStackSample},
	}
	for _, tc := range cases {
		if got := normalizeStudioStack(tc.in); got != tc.want {
			t.Fatalf("normalizeStudioStack(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestListStudioStacks_twoTiers(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip("no project root")
	}
	stacks, err := listStudioStacks(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 2 {
		t.Fatalf("want 2 studio tiers, got %d: %+v", len(stacks), stacks)
	}
	if stacks[0].Label != "样片" || stacks[0].CostMode != "cap" {
		t.Fatalf("first tier: %+v", stacks[0])
	}
	if stacks[1].Label != "成片" || stacks[1].CostMode != "per_30_sec" {
		t.Fatalf("second tier: %+v", stacks[1])
	}
}

func TestEffectiveStackProfile_normalizesLegacy(t *testing.T) {
	prefs := &userPrefs{StackProfile: "micro-movie-wan-flash"}
	if got := effectiveStackProfile(prefs, ""); got != StudioStackFinal {
		t.Fatalf("legacy prefs got %q", got)
	}
	if got := effectiveStackProfile(prefs, "micro-movie-wan-flash"); got != StudioStackFinal {
		t.Fatalf("override got %q", got)
	}
}
