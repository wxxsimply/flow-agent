package vault

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
)

func TestSearchFTS(t *testing.T) {
	dir := t.TempDir()
	seriesDir := filepath.Join(dir, "demo")
	if err := os.MkdirAll(filepath.Join(seriesDir, "vault"), 0o755); err != nil {
		t.Fatal(err)
	}
	bible := `series_id: demo
title: 雨夜复仇
logline: 女主雨夜接到电话`
	if err := os.WriteFile(filepath.Join(seriesDir, "series-bible.yaml"), []byte(bible), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seriesDir, "foreshadows.yaml"), []byte("foreshadows:\n  - note: 伏笔 未寄出的信\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	app := &config.App{SeriesDir: dir}
	v := ForSeries(app, "demo")
	if err := v.SyncIndex(); err != nil {
		t.Fatal(err)
	}
	hits, err := v.SearchFTS("demo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected hits for demo")
	}

	// 中文子串搜索（LIKE 回退）
	hits2, err := v.SearchFTS("雨夜", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits2) == 0 {
		t.Fatal("expected hits for 雨夜")
	}

	hits3, err := v.SearchFTS("伏笔", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits3) == 0 {
		t.Fatal("expected hits for 伏笔")
	}
}
