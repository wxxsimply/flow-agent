package artifacts

import "testing"

func TestContinuityRecount(t *testing.T) {
	r := &ContinuityReport{
		Issues: []ContinuityIssue{
			{Severity: "critical", SceneID: 2},
			{Severity: "warning"},
		},
	}
	r.Recount()
	if r.CriticalCount != 1 || r.WarningCount != 1 || r.Passed {
		t.Fatalf("got critical=%d warning=%d passed=%v", r.CriticalCount, r.WarningCount, r.Passed)
	}
	ids := r.CriticalSceneIDs()
	if len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("scene ids: %v", ids)
	}
}

func TestSceneIDsForRewriteNextScene(t *testing.T) {
	r := &ContinuityReport{
		Issues: []ContinuityIssue{{
			Severity:   "critical",
			SceneID:    4,
			Suggestion: "应在 Scene 4 结尾或 Scene 5 开头加入心理留白",
		}},
	}
	valid := map[int]bool{4: true, 5: true}
	ids := r.SceneIDsForRewrite(valid)
	if len(ids) != 2 {
		t.Fatalf("want scenes 4,5 got %v", ids)
	}
}
