package compliance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWordLists(t *testing.T) {
	dir := t.TempDir()
	comp := filepath.Join(dir, "compliance")
	if err := os.MkdirAll(comp, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "words.txt"), []byte("# custom\n赌博\nwarn:香烟\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(comp, "platform-douyin.txt"), []byte("block:毒品\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := LoadWordLists(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("want 3 entries, got %d", len(entries))
	}
}

func TestScanSourcesBlockAndWarn(t *testing.T) {
	entries := []WordEntry{
		{Term: "赌博", Severity: "block"},
		{Term: "香烟", Severity: "warning"},
	}
	report := ScanSources(entries, []TextSource{
		{Name: "chapter.md", Text: "他去了赌博网站。"},
		{Name: "storyboard.shots[0].subtitle", Text: "别碰香烟"},
	})
	if !report.Blocked || report.BlockCount != 1 || report.WarningCount != 1 {
		t.Fatalf("blocked=%v blocks=%d warnings=%d", report.Blocked, report.BlockCount, report.WarningCount)
	}
	if report.Blocks[0].Word != "赌博" || report.Warnings[0].Word != "香烟" {
		t.Fatalf("blocks=%v warnings=%v", report.Blocks, report.Warnings)
	}
}

func TestScanSourcesClean(t *testing.T) {
	entries := []WordEntry{{Term: "赌博", Severity: "block"}}
	report := ScanSources(entries, []TextSource{{Name: "chapter.md", Text: "雨夜诀别的故事。"}})
	if report.Blocked || len(report.Blocks) > 0 {
		t.Fatalf("unexpected block: %+v", report)
	}
}

func TestScanSourcesASCIICaseInsensitive(t *testing.T) {
	entries := []WordEntry{{Term: "Casino", Severity: "block"}}
	report := ScanSources(entries, []TextSource{{Name: "chapter.md", Text: "visit the CASINO tonight"}})
	if !report.Blocked {
		t.Fatal("expected block for case-insensitive ASCII match")
	}
}
