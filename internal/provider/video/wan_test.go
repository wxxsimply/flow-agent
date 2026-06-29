package video

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestWanImageToVideo_MockServer(t *testing.T) {
	videoURL := ""
	var pollCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == wanVideoSynthesisPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output": map[string]string{"task_id": "task-mock-1", "task_status": "PENDING"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/tasks/task-mock-1":
			pollCount++
			if pollCount < 2 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"output": map[string]string{"task_id": "task-mock-1", "task_status": "RUNNING"},
				})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"output": map[string]string{
					"task_id":     "task-mock-1",
					"task_status": "SUCCEEDED",
					"video_url":   videoURL,
				},
			})
		case r.URL.Path == "/clip.mp4":
			w.Write([]byte("mock-mp4-bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	videoURL = srv.URL + "/clip.mp4"
	defer srv.Close()

	dir := t.TempDir()
	imgPath := filepath.Join(dir, "frame.png")
	if err := os.WriteFile(imgPath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}

	p := config.Providers{DashScope: config.DashScopeConfig{APIKey: "sk-test", Region: "cn-beijing"}}
	w := NewWan(p, "wan2.6-i2v-flash", "", "720P", true, false)
	w.baseURL = srv.URL
	w.http = srv.Client()
	w.pollHTTP = srv.Client()

	out, err := w.ImageToVideo(context.Background(), ImageToVideoRequest{
		ImagePath:   imgPath,
		Prompt:      "镜头推进",
		DurationSec: 5,
	})
	if err != nil {
		t.Fatalf("ImageToVideo: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "mock-mp4-bytes" {
		t.Fatalf("got %q", data)
	}
}
