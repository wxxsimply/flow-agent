package artifacts

import (
	"encoding/json"
	"testing"
)

func TestFlexStringUnmarshal(t *testing.T) {
	var s FlexString
	if err := json.Unmarshal([]byte(`"whoosh"`), &s); err != nil || s.String() != "whoosh" {
		t.Fatalf("string: %q err=%v", s, err)
	}
	if err := json.Unmarshal([]byte(`["雨声","BGM"]`), &s); err != nil || s.String() != "雨声, BGM" {
		t.Fatalf("array: %q err=%v", s, err)
	}
}

func TestStoryboardUnmarshalSfxArray(t *testing.T) {
	raw := `{"episode_no":1,"target_duration_sec":180,"shots":[{"id":"s01","duration_sec":10,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"a","subtitle":"a","sfx":["雨声"]}]}`
	var sb Storyboard
	if err := json.Unmarshal([]byte(raw), &sb); err != nil {
		t.Fatal(err)
	}
	if sb.Shots[0].SFX.String() != "雨声" {
		t.Fatalf("sfx=%q", sb.Shots[0].SFX)
	}
}

func TestStoryboardUnmarshalHeldPropsArray(t *testing.T) {
	raw := `{"episode_no":1,"target_duration_sec":120,"shots":[{"id":"s01","duration_sec":10,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"a","subtitle":"a","held_props":["右手单手持剑"]}]}`
	var sb Storyboard
	if err := json.Unmarshal([]byte(raw), &sb); err != nil {
		t.Fatal(err)
	}
	if sb.Shots[0].HeldProps.String() != "右手单手持剑" {
		t.Fatalf("held_props=%q", sb.Shots[0].HeldProps)
	}
}
