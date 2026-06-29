package artifacts

import "testing"

func TestNarrationComplete(t *testing.T) {
	if !NarrationComplete("雨夜，天台边缘。") {
		t.Fatal("expected complete")
	}
	if NarrationComplete("雨夜，天台边缘，少年紧握手机，霓") {
		t.Fatal("expected incomplete")
	}
}

func TestIncompleteNarrationShots(t *testing.T) {
	sb := &Storyboard{
		Shots: []Shot{
			{ID: "s01", Narration: "完整句。"},
			{ID: "s02", Narration: "截断句，"},
		},
	}
	ids := sb.IncompleteNarrationShots()
	if len(ids) != 1 || ids[0] != "s02" {
		t.Fatalf("got %v", ids)
	}
}

func TestScenesSimilar(t *testing.T) {
	a := "雨夜天台，霓虹灯光冷蓝与暗红交织"
	b := "雨夜天台，霓虹冷蓝暗红，栏杆挂水珠"
	if !ScenesSimilar(a, b) {
		t.Fatal("expected similar scenes")
	}
	if ScenesSimilar(a, "城堡大厅，壁炉火焰") {
		t.Fatal("expected different scenes")
	}
}

func TestSceneChanged(t *testing.T) {
	prev := Shot{SceneBackground: "雨夜天台，霓虹"}
	curr := Shot{SceneBackground: "雨夜天台，铁门"}
	if SceneChanged(prev, curr) {
		t.Fatal("same location should not scene-change")
	}
	next := Shot{SceneBackground: "城堡大厅，壁炉"}
	if !SceneChanged(curr, next) {
		t.Fatal("different location should scene-change")
	}
}
