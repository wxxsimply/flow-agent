package runner

import "testing"

func TestEvalDurationCondition(t *testing.T) {
	if !evalDurationCondition(180, 180, 3) {
		t.Fatal("exact match")
	}
	if evalDurationCondition(200, 180, 3) {
		t.Fatal("should fail")
	}
}
