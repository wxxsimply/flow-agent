package artifacts

import (
	"encoding/json"
	"testing"
)

func TestHookPlanFlexibleSceneID(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []int
	}{
		{
			name: "int_ids",
			raw:  `{"episode_no":1,"hook_type":"x","hook_line":"y","scene_count":2,"scenes":[{"id":1,"title":"a","goal":"b","max_chars":200},{"id":2,"title":"c","goal":"d","max_chars":200}]}`,
			want: []int{1, 2},
		},
		{
			name: "string_numeric_ids",
			raw:  `{"episode_no":1,"hook_type":"x","hook_line":"y","scene_count":2,"scenes":[{"id":"1","title":"a","goal":"b","max_chars":"200"},{"id":"2","title":"c","goal":"d","max_chars":"200"}]}`,
			want: []int{1, 2},
		},
		{
			name: "string_prefixed_ids",
			raw:  `{"episode_no":1,"hook_type":"x","hook_line":"y","scene_count":2,"scenes":[{"id":"scene-1","title":"a","goal":"b"},{"id":"S02","title":"c","goal":"d"}]}`,
			want: []int{1, 2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var h HookPlan
			if err := json.Unmarshal([]byte(c.raw), &h); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if len(h.Scenes) != len(c.want) {
				t.Fatalf("scene count = %d, want %d", len(h.Scenes), len(c.want))
			}
			for i, w := range c.want {
				if h.Scenes[i].ID != w {
					t.Errorf("scene[%d].ID = %d, want %d", i, h.Scenes[i].ID, w)
				}
			}
		})
	}
}
