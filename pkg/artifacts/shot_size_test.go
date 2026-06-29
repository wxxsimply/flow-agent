package artifacts

import "testing"

func TestNormalizeShotSize(t *testing.T) {
	if NormalizeShotSize("特写") != ShotSizeClose {
		t.Fatal("特写 -> close")
	}
	if NormalizeShotSize("") != ShotSizeMedium {
		t.Fatal("empty -> medium")
	}
}

func TestShotSizePromptHint(t *testing.T) {
	h := ShotSizePromptHint(ShotSizeWide)
	if h == "" || len(h) < 4 {
		t.Fatal("wide hint empty")
	}
}
