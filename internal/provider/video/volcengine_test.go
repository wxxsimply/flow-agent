package video

import "testing"

func TestParseTaskID(t *testing.T) {
	id, err := parseTaskID([]byte(`{"id":"cgt-2025-test"}`))
	if err != nil || id != "cgt-2025-test" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestParseTaskResult(t *testing.T) {
	st, url, _, err := parseTaskResult([]byte(`{"status":"succeeded","content":{"video_url":"https://example.com/a.mp4"}}`))
	if err != nil || st != "succeeded" || url == "" {
		t.Fatalf("st=%q url=%q err=%v", st, url, err)
	}
}
