package artifacts

import "testing"

func TestNormalizeSeverityDowngradesStyle(t *testing.T) {
	r := &ContinuityReport{
		Issues: []ContinuityIssue{{
			Severity: "critical",
			Category: "character_state",
			Message:  "Scene 4 心理留白不足，隐忍特质未体现过渡",
		}},
	}
	r.NormalizeSeverity()
	if r.CriticalCount != 0 || r.WarningCount != 1 {
		t.Fatalf("critical=%d warning=%d", r.CriticalCount, r.WarningCount)
	}
}

func TestNormalizeSeverityKeepsFactual(t *testing.T) {
	r := &ContinuityReport{
		Issues: []ContinuityIssue{{
			Severity: "critical",
			Category: "character_state",
			Message:  "与 bible 设定矛盾：林晚从未认识顾沉",
		}},
	}
	r.NormalizeSeverity()
	if r.CriticalCount != 1 {
		t.Fatalf("want 1 critical got %d", r.CriticalCount)
	}
}
