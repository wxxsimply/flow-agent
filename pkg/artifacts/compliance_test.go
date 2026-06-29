package artifacts

import "testing"

func TestComplianceRecount(t *testing.T) {
	r := &ComplianceReport{
		Blocks: []ComplianceIssue{
			{Severity: "block", Word: "赌博", Source: "chapter.md"},
		},
		Warnings: []ComplianceIssue{
			{Severity: "warning", Word: "香烟", Source: "storyboard.shots[0].subtitle"},
		},
	}
	r.Recount()
	if !r.Blocked || r.BlockCount != 1 || r.WarningCount != 1 {
		t.Fatalf("got blocked=%v block=%d warning=%d", r.Blocked, r.BlockCount, r.WarningCount)
	}
}

func TestComplianceRecountClean(t *testing.T) {
	r := &ComplianceReport{}
	r.Recount()
	if r.Blocked || r.BlockCount != 0 || r.WarningCount != 0 {
		t.Fatalf("got blocked=%v block=%d warning=%d", r.Blocked, r.BlockCount, r.WarningCount)
	}
}
