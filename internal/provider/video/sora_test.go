package video

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestSora_ImageToVideo(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "frame.png")
	if err := os.WriteFile(imgPath, png, 0o644); err != nil {
		t.Fatal(err)
	}

	var pollCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/videos"):
			if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Errorf("create videos Content-Type = %q, want application/json", ct)
			}
			var req struct {
				InputReference *struct {
					ImageURL string `json:"image_url"`
				} `json:"input_reference"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode create body: %v", err)
			} else if req.InputReference == nil || !strings.HasPrefix(req.InputReference.ImageURL, "data:image/png;base64,") {
				t.Errorf("input_reference image_url missing or wrong prefix: %+v", req.InputReference)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"video_test123","status":"queued"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/videos/video_test123/content"):
			_, _ = w.Write([]byte("fake-sora-mp4"))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/videos/video_test123"):
			pollCount++
			w.Header().Set("Content-Type", "application/json")
			if pollCount < 2 {
				_, _ = w.Write([]byte(`{"id":"video_test123","status":"in_progress"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"video_test123","status":"completed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewSora(config.Providers{
		OpenAI: config.OpenAIConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1"},
	}, "sora-2", "9:16", "720x1280", true)

	out, err := client.ImageToVideo(context.Background(), ImageToVideoRequest{
		ImagePath:   imgPath,
		Prompt:      "slow pan portrait",
		DurationSec: 5,
	})
	if err != nil {
		t.Fatalf("ImageToVideo: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-sora-mp4" {
		t.Fatalf("got %q", data)
	}
}

func TestClampSoraDuration(t *testing.T) {
	if clampSoraDuration(3) != "4" {
		t.Fatalf("3 -> %s", clampSoraDuration(3))
	}
	if clampSoraDuration(5) != "8" {
		t.Fatalf("5 -> %s", clampSoraDuration(5))
	}
	if clampSoraDuration(12) != "12" {
		t.Fatalf("12 -> %s", clampSoraDuration(12))
	}
	if clampSoraDuration(30) != "12" {
		t.Fatalf("30 -> %s", clampSoraDuration(30))
	}
}

func TestNormalizeSoraSize(t *testing.T) {
	if got := normalizeSoraSize("", "9:16", "sora-2"); got != "720x1280" {
		t.Fatalf("got %s", got)
	}
	if got := normalizeSoraSize("", "9:16", "sora-2-pro"); got != "1080x1920" {
		t.Fatalf("got %s", got)
	}
}

func TestSora_createJobError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "invalid api key"}})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "frame.png")
	if err := os.WriteFile(imgPath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewSora(config.Providers{
		OpenAI: config.OpenAIConfig{APIKey: "bad", BaseURL: srv.URL + "/v1"},
	}, "", "9:16", "720x1280", false)
	_, err := client.ImageToVideo(context.Background(), ImageToVideoRequest{
		ImagePath: imgPath,
		Prompt:    "test",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("expected error, got %v", err)
	}
}
