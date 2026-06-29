package agent

import (
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestShotKeyframeImagePromptUsesBeatAndPhysics(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{Stack: &config.Stack{Name: "micro-movie-wan-flash"}},
	}
	shot := artifacts.Shot{
		ID:               "s01",
		Expanded:         true,
		ShotSize:         "medium",
		PhysicsCues:      "重力向下，鞋底贴地",
		ForbiddenPhysics: "穿模",
		ActionBeats:      []string{"预备站姿", "伸手", "收手"},
	}
	p1 := ShotKeyframeImagePrompt(rc, shot, shot.ActionBeats[0], nil, false)
	p2 := ShotKeyframeImagePrompt(rc, shot, shot.ActionBeats[1], nil, false)
	if p1 == p2 {
		t.Fatal("keyframes should differ by beat")
	}
	if !strings.Contains(p1, "预备站姿") {
		t.Fatal("missing beat text")
	}
	if !strings.Contains(p1, "物理约束") || !strings.Contains(p1, "重力") {
		t.Fatal("missing physics_cues in keyframe prompt")
	}
}

func TestShotSegmentMotionPromptIncludesPhysics(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{Stack: &config.Stack{Name: "micro-movie-wan-flash"}},
	}
	shot := artifacts.Shot{
		VisualPrompt: "雨夜街道",
		PhysicsCues:  "雨滴竖直下落",
		Expanded:     true,
		ShotSize:     "wide",
	}
	vidCfg := config.StackVideoConfig{MotionPromptSuffix: "，测试后缀"}
	m := ShotSegmentMotionPrompt(rc, vidCfg, shot, "预备", "进行")
	if !strings.Contains(m, "物理约束") && !strings.Contains(m, "雨滴") {
		t.Fatalf("expected physics in motion: %s", m)
	}
	if !strings.Contains(m, "预备") || !strings.Contains(m, "进行") {
		t.Fatal("missing transition beats")
	}
}
