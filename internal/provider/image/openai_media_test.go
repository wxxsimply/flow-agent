package image

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIMedia_Generate_urlResponse(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47}
	imgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(png)
	}))
	defer imgSrv.Close()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/images/generations") {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer sk-test") {
			t.Fatalf("auth header = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"url":"` + imgSrv.URL + `/img.png"}]}`))
	}))
	defer srv.Close()

	client := NewOpenAIMedia("sk-test", srv.URL, "dall-e-3")
	out, err := client.Generate(context.Background(), GenerateRequest{Prompt: "a cat", Width: 1024, Height: 1024})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(png) {
		t.Fatalf("got %d bytes, want png header", len(out))
	}
}

func TestOpenAIMedia_Generate_b64Response(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47}
	b64 := base64.StdEncoding.EncodeToString(png)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"b64_json":"` + b64 + `"}]}`))
	}))
	defer srv.Close()

	client := NewOpenAIMedia("sk-test", srv.URL, "dall-e-3")
	out, err := client.Generate(context.Background(), GenerateRequest{Prompt: "a dog"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(png) {
		t.Fatal("b64 decode mismatch")
	}
}

func TestOpenAIMedia_notConfigured(t *testing.T) {
	client := NewOpenAIMedia("", "", "")
	if client != nil {
		t.Fatal("expected nil client without key")
	}
}
