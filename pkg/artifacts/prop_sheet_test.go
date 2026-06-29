package artifacts

import (
	"strings"
	"testing"
)

func TestPropViewLockBlock(t *testing.T) {
	sheets := &PropSheets{
		Props: []PropSheetEntry{{
			ID:         "p01-dagger",
			Name:       "匕首",
			Appearance: "银质双刃，皮革握柄",
			TurnaroundViews: &CharacterViewPaths{
				Front: "artifacts/assets/props/p01-dagger-front.png",
				Side:  "artifacts/assets/props/p01-dagger-side.png",
			},
		}},
	}
	block := sheets.PropViewLockBlock("中景，匕首横于胸前", []string{"p01-dagger"})
	if block == "" {
		t.Fatal("expected block")
	}
	if !strings.Contains(block, "[PROP_VIEW_LOCK]") {
		t.Fatalf("missing tag: %s", block)
	}
	if !strings.Contains(block, "正面") || !strings.Contains(block, "匕首") {
		t.Fatalf("block=%s", block)
	}
	if !strings.Contains(block, "正面分图已生成") {
		t.Fatalf("expected front view hint: %s", block)
	}

	sideBlock := sheets.PropViewLockBlock("侧面，匕首出鞘", []string{"p01-dagger"})
	if !strings.Contains(sideBlock, "侧面") {
		t.Fatalf("side block=%s", sideBlock)
	}
}

func TestPropsForShot_PropRefs(t *testing.T) {
	sheets := &PropSheets{
		Props: []PropSheetEntry{
			{ID: "p01-dagger", Name: "匕首", Appearance: "银质"},
			{ID: "p02-shield", Name: "圆盾", Appearance: "木质"},
		},
	}
	shot := Shot{
		PropRefs:  []string{"p01-dagger", "p02-shield"},
		HeldProps: FlexString("右手：匕首；左手：圆盾"),
	}
	props := sheets.PropsForShot(shot)
	if len(props) != 2 {
		t.Fatalf("got %d props", len(props))
	}
}

func TestPropAppearanceBlock(t *testing.T) {
	sheets := &PropSheets{
		Props: []PropSheetEntry{{
			ID: "p01-dagger", Name: "匕首", Appearance: "银质",
			TurnaroundViews: &CharacterViewPaths{Front: "f.png"},
		}},
	}
	block := sheets.AppearanceBlock(sheets.Props)
	if !strings.Contains(block, "[PROPS]") || !strings.Contains(block, "匕首") {
		t.Fatalf("block=%s", block)
	}
}
