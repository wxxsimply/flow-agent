package artifacts

import (
	"encoding/json"
	"testing"
)

func TestShotUnmarshalNumericID(t *testing.T) {
	raw := `{"id":1,"duration_sec":5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"a","subtitle":"a"}`
	var sh Shot
	if err := json.Unmarshal([]byte(raw), &sh); err != nil {
		t.Fatal(err)
	}
	if sh.ID != "s01" {
		t.Fatalf("id=%q want s01", sh.ID)
	}
}

func TestShotUnmarshalStringNumericID(t *testing.T) {
	raw := `{"id":"3","duration_sec":5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"a","subtitle":"a"}`
	var sh Shot
	if err := json.Unmarshal([]byte(raw), &sh); err != nil {
		t.Fatal(err)
	}
	if sh.ID != "s03" {
		t.Fatalf("id=%q want s03", sh.ID)
	}
}

func TestStoryboardUnmarshalNumericShotIDs(t *testing.T) {
	raw := `{"episode_no":1,"target_duration_sec":45,"shots":[{"id":1,"duration_sec":5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"a","subtitle":"a"},{"id":2,"duration_sec":5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"y","narration":"b","subtitle":"b"}]}`
	var sb Storyboard
	if err := json.Unmarshal([]byte(raw), &sb); err != nil {
		t.Fatal(err)
	}
	if sb.Shots[0].ID != "s01" || sb.Shots[1].ID != "s02" {
		t.Fatalf("ids=%q %q", sb.Shots[0].ID, sb.Shots[1].ID)
	}
}
