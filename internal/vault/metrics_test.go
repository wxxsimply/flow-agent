package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestPublishMetricsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	app := &config.App{SeriesDir: dir}
	v := ForSeries(app, "test-series")
	m := &artifacts.PublishMetrics{
		EpisodeNo:       1,
		Views24h:        12000,
		CompletionRate:  0.42,
		CommentKeywords: []string{"反转", "爽"},
	}
	if err := v.SavePublishMetrics(m); err != nil {
		t.Fatal(err)
	}
	got, err := v.LoadPublishMetrics(1)
	if err != nil || got == nil || got.Views24h != 12000 {
		t.Fatalf("load: %+v err=%v", got, err)
	}
	prev, _ := v.LoadPreviousPublishMetrics(2)
	if prev == nil || prev.Views24h != 12000 {
		t.Fatal("expected prev ep1 metrics for ep2")
	}
	if err := v.SaveNextEpisodeHints(2, "# hints for ep2"); err != nil {
		t.Fatal(err)
	}
	if h := v.LoadNextEpisodeHints(2); h == "" {
		t.Fatal("hints missing")
	}
	if _, err := os.Stat(filepath.Join(dir, "test-series", "publish-metrics", "episode-001.json")); err != nil {
		t.Fatal(err)
	}
	eps, err := v.ListPublishMetricsEpisodes()
	if err != nil || len(eps) != 1 || eps[0] != 1 {
		t.Fatalf("list: %v err=%v", eps, err)
	}
}
