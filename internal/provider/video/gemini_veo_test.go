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

func TestGeminiVeo_ImageToVideo(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "frame.png")
	if err := os.WriteFile(imgPath, png, 0o644); err != nil {
		t.Fatal(err)
	}

	var pollCount int
	var videoURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "predictLongRunning"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"operations/test-op"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "operations/test-op"):
			pollCount++
			w.Header().Set("Content-Type", "application/json")
			if pollCount < 2 {
				_, _ = w.Write([]byte(`{"done":false}`))
				return
			}
			resp := map[string]any{
				"done": true,
				"response": map[string]any{
					"generateVideoResponse": map[string]any{
						"generatedSamples": []map[string]any{
							{"video": map[string]any{"uri": videoURL}},
						},
					},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "clip.mp4"):
			_, _ = w.Write([]byte("fake-mp4"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	videoURL = srv.URL + "/clip.mp4"

	client := NewGeminiVeo(config.Providers{
		Gemini: config.GeminiConfig{APIKey: "test-key", BaseURL: srv.URL + "/v1beta"},
	}, "veo-3.1-lite-generate-preview", "9:16", "720p", true)

	out, err := client.ImageToVideo(context.Background(), ImageToVideoRequest{
		ImagePath:   imgPath,
		Prompt:      "slow pan, portrait 9:16",
		DurationSec: 5,
	})
	if err != nil {
		t.Fatalf("ImageToVideo: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake-mp4" {
		t.Fatalf("got %q", data)
	}
}

func TestClampVeoDuration(t *testing.T) {
	if clampVeoDuration(3) != "4" {
		t.Fatalf("3 -> %s", clampVeoDuration(3))
	}
	if clampVeoDuration(5) != "6" {
		t.Fatalf("5 -> %s", clampVeoDuration(5))
	}
	if clampVeoDuration(10) != "8" {
		t.Fatalf("10 -> %s", clampVeoDuration(10))
	}
}

func TestNormalizeVeoResolution(t *testing.T) {
	if normalizeVeoResolution("720P") != "720p" {
		t.Fatalf("720P")
	}
}

func TestGeminiVeo_createOperationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"message": "bad key"}})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "frame.png")
	if err := os.WriteFile(imgPath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o644); err != nil {
		t.Fatal(err)
	}

	client := NewGeminiVeo(config.Providers{
		Gemini: config.GeminiConfig{APIKey: "bad", BaseURL: srv.URL + "/v1beta"},
	}, "", "", "", false)
	_, err := client.ImageToVideo(context.Background(), ImageToVideoRequest{
		ImagePath: imgPath,
		Prompt:    "test",
	})
	if err == nil || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("expected error, got %v", err)
	}
}
