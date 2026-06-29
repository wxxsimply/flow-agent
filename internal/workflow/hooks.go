package workflow

import "fmt"

// StageHook 单个 hook 触发点（如 before_plan）及动作列表。
type StageHook struct {
	Timing  string   // before_plan | after_learn | …
	Actions []string // inject_l0_series_bible 等
}

// ParseStageHooks 解析 YAML hooks 字段（list of single-key maps）。
func ParseStageHooks(raw any) []StageHook {
	if raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var out []StageHook
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			if m2, ok2 := item.(map[interface{}]interface{}); ok2 {
				m = map[string]any{}
				for k, v := range m2 {
					m[fmt.Sprint(k)] = v
				}
			} else {
				continue
			}
		}
		for timing, actionsRaw := range m {
			actions := stringSlice(actionsRaw)
			if len(actions) == 0 {
				continue
			}
			out = append(out, StageHook{Timing: timing, Actions: actions})
		}
	}
	return out
}

// HooksForPhase 筛选 before_<stage> / after_<stage> 的 hooks。
func HooksForPhase(hooks []StageHook, phase, stageID string) []StageHook {
	prefix := phase + "_" + stageID
	var out []StageHook
	for _, h := range hooks {
		if h.Timing == prefix {
			out = append(out, h)
		}
	}
	return out
}

func stringSlice(v any) []string {
	switch a := v.(type) {
	case []string:
		return a
	case []any:
		out := make([]string, 0, len(a))
		for _, x := range a {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
