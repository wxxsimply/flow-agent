package agent

import (
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestCrossShotContinuitySuffixSameScene(t *testing.T) {
	prev := artifacts.Shot{
		ID:              "s01",
		SceneBackground: "霓虹雨夜街道",
		ActionBeats:     []string{"起始", "进行", "骑士转身离去"},
	}
	curr := artifacts.Shot{ID: "s02", SceneBackground: "霓虹雨夜街道天桥"}
	got := crossShotContinuitySuffix(prev, curr)
	if strings.Contains(got, "转身离去") {
		t.Fatalf("soft suffix should not bind end beat: %q", got)
	}
	if !strings.Contains(got, "同一场景") {
		t.Fatalf("expected same-scene hint: %q", got)
	}
}

func TestSceneChangedAllowsHardCut(t *testing.T) {
	prev := artifacts.Shot{ID: "s01", SceneBackground: "雨夜霓虹街道"}
	curr := artifacts.Shot{ID: "s02", SceneBackground: "室内废弃工厂"}
	if !sceneChanged(prev, curr) {
		t.Fatal("expected scene change")
	}
	got := crossShotContinuitySuffix(prev, curr)
	if !strings.Contains(got, "硬切") {
		t.Fatalf("expected hard cut hint: %q", got)
	}
	if shouldLinkToPreviousShot(prev, curr) {
		t.Fatal("should not link prev shot on scene change")
	}
}

func TestShouldLinkSameScene(t *testing.T) {
	prev := artifacts.Shot{ID: "s01", SceneBackground: "天台夜景"}
	curr := artifacts.Shot{ID: "s02", SceneBackground: "天台夜景近景"}
	if !shouldLinkToPreviousShot(prev, curr) {
		t.Fatal("expected link for same scene")
	}
}
