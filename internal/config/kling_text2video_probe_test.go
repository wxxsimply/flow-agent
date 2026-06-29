package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeKlingText2VideoModelsLive(t *testing.T) {
	if os.Getenv("KLING_LIVE_PROBE") == "" {
		t.Skip("set KLING_LIVE_PROBE=1 to run")
	}
	root, err := FindRoot()
	if err != nil {
		t.Fatal(err)
	}
	p, err := LoadProviders(filepath.Join(root, "config", "providers.local.yaml"))
	if err != nil || !p.KlingEnabled() {
		t.Skip("kling not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	results := ProbeKlingText2VideoModels(ctx, p, p.Kling.BaseURL)
	t.Log(FormatKlingText2VideoProbeReport(results))
	for _, r := range results {
		if r.OK {
			return
		}
	}
	t.Fatal("no working text2video model")
}
