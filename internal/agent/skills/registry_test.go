package skills

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestLoadFromRoot(t *testing.T) {
	root, err := config.FindRoot()
	if err != nil {
		t.Skip(err)
	}
	reg, err := LoadFromRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Skills) == 0 {
		t.Fatal("expected project skills")
	}
	if reg.Skills["micro-movie-director"] == nil {
		t.Fatal("missing micro-movie-director")
	}
}

func TestInjectSystemContainsCameraLanguage(t *testing.T) {
	out := InjectSystem("BASE", StageGenerateShots)
	if !strings.Contains(out, "BASE") {
		t.Fatal("base missing")
	}
	if !strings.Contains(out, "camera-language.md") && !strings.Contains(out, "景别") {
		t.Fatal("expected camera-language content or fallback")
	}
}

func TestInjectSystemTruncatesLongStage(t *testing.T) {
	reg, err := Default()
	if err != nil || reg == nil {
		t.Skip("no registry")
	}
	var b strings.Builder
	reg.blockForStage(StageExpandBriefSegment, &b)
	runes := utf8.RuneCountInString(b.String())
	max := stageMaxRefRunes[StageExpandBriefSegment] + 1200 // skill headers
	if runes > max {
		t.Fatalf("segment refs too long: %d runes (max ~%d)", runes, max)
	}
}

func TestMotionPromptBlockMultipleBullets(t *testing.T) {
	block := MotionPromptBlock()
	if !strings.HasPrefix(block, "，") {
		t.Fatalf("expected leading comma block: %q", block)
	}
	parts := strings.Split(strings.TrimPrefix(block, "，"), "，")
	if len(parts) < 3 {
		t.Fatalf("expected >=3 motion bullets, got %d: %s", len(parts), block)
	}
}

func TestExtractBulletsAfterHeading(t *testing.T) {
	md := "## 正向\n\n- 第一句\n- 第二句\n\n## 负向\n\n- 禁止第三句\n"
	pos := extractBulletsAfterHeading(md, "正向", 5)
	if len(pos) != 2 || pos[0] != "第一句" {
		t.Fatalf("pos got %v", pos)
	}
	neg := extractBulletsAfterHeading(md, "负向", 5)
	if len(neg) != 1 || neg[0] != "禁止第三句" {
		t.Fatalf("neg got %v", neg)
	}
}

func TestExtractBullets(t *testing.T) {
	md := "# t\n\n- 第一句约束\n- 第二句\n"
	b := extractBullets(md, 5)
	if len(b) != 2 {
		t.Fatalf("got %v", b)
	}
}

func TestTruncateRunesWithConsumed(t *testing.T) {
	s := "一二三四五六七八九十"
	out, n := truncateRunesWithConsumed(s, 5)
	if n != 5 || utf8.RuneCountInString(out) != 5 {
		t.Fatalf("got %q n=%d", out, n)
	}
}

func TestReportIncludesRefsByStage(t *testing.T) {
	rep := Report()
	if rep.Refs == nil {
		t.Fatal("expected refs_by_stage")
	}
	refs := rep.Refs[string(StageGenerateShots)]
	found := false
	for _, r := range refs {
		if r == "camera-language.md" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("generate_shots refs missing camera-language: %v", refs)
	}
}
