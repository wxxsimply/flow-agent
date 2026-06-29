package artifacts

import (
	"strings"
	"testing"
)

func TestTurnaroundViewForPrompt(t *testing.T) {
	if got := TurnaroundViewForPrompt("骑士背影走向城门"); got != "back" {
		t.Fatalf("back: %q", got)
	}
	if got := TurnaroundViewForPrompt("侧面跟拍"); got != "side" {
		t.Fatalf("side: %q", got)
	}
	if got := TurnaroundViewForPrompt("正面特写"); got != "front" {
		t.Fatalf("front: %q", got)
	}
}

func TestViewLockBlock(t *testing.T) {
	sheets := &CharacterSheets{
		Characters: []CharacterSheetEntry{{
			Name:       "骑士",
			Appearance: "破损铠甲",
			TurnaroundViews: &CharacterViewPaths{
				Front: "a/front.png",
				Back:  "a/back.png",
			},
		}},
	}
	block := sheets.ViewLockBlock("他背对镜头离开")
	if !strings.Contains(block, "背面") || !strings.Contains(block, "破损铠甲") {
		t.Fatalf("block=%q", block)
	}
}

func TestViewLockBlock_nonProtagonistShot(t *testing.T) {
	sheets := &CharacterSheets{
		Characters: []CharacterSheetEntry{{
			Name:       "主角",
			Appearance: "国王，深红长袍，四十岁左右",
		}},
	}
	block := sheets.ViewLockBlock("臣子面部特写，深灰长袍，双手奉羊皮纸")
	if strings.Contains(block, "国王") {
		t.Fatalf("should not lock king on minister shot: %q", block)
	}
	if !strings.Contains(block, "臣子") && !strings.Contains(block, "画面描述") {
		t.Fatalf("expected visual-based lock: %q", block)
	}
}
