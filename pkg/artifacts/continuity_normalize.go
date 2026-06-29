package artifacts

import "strings"

// factualContradictionKeywords 仅含此类表述时才保留 character_state 为 critical。
var factualContradictionKeywords = []string{
	"违背", "矛盾", "不符合", "错误", "不可能", "违反", "从未", "已经死亡",
	"时间线", "穿帮", "设定冲突", "与 bible", "与设定",
}

// NormalizeSeverity 将文风/节奏类 character_state 误报降为 warning。
func (r *ContinuityReport) NormalizeSeverity() {
	for i := range r.Issues {
		if r.Issues[i].Severity != "critical" {
			continue
		}
		if shouldDowngradeToWarning(r.Issues[i]) {
			r.Issues[i].Severity = "warning"
		}
	}
	r.Recount()
}

func shouldDowngradeToWarning(iss ContinuityIssue) bool {
	msg := iss.Message + " " + iss.Suggestion
	switch iss.Category {
	case "character_state", "pacing", "style", "emotion":
		return !containsAny(msg, factualContradictionKeywords)
	case "":
		// 未分类：含心理/留白/隐忍/过渡等多为文风建议
		styleHints := []string{"心理", "留白", "隐忍", "过渡", "节奏", "格调", "即时性", "瓦解", "特质", "留白"}
		if containsAny(msg, styleHints) && !containsAny(msg, factualContradictionKeywords) {
			return true
		}
	}
	return false
}

// SoftenRemainingCriticals AutoGate 下将仍剩的 critical 全部降为 warning（开发放行）。
func (r *ContinuityReport) SoftenRemainingCriticals() {
	for i := range r.Issues {
		if r.Issues[i].Severity == "critical" {
			r.Issues[i].Severity = "warning"
		}
	}
	r.Recount()
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
