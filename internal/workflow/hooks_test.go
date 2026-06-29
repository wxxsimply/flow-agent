package workflow

import "testing"

func TestParseStageHooks(t *testing.T) {
	raw := []any{
		map[string]any{
			"before_plan": []any{"inject_l0_series_bible", "inject_publish_metrics"},
		},
		map[string]any{
			"after_learn": []any{"archive_episode_to_series_vault"},
		},
	}
	hooks := ParseStageHooks(raw)
	if len(hooks) != 2 {
		t.Fatalf("got %d hooks", len(hooks))
	}
	before := HooksForPhase(hooks, "before", "plan")
	if len(before) != 1 || len(before[0].Actions) != 2 {
		t.Fatalf("before plan: %+v", before)
	}
}
