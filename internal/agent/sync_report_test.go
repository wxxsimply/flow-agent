package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestBuildSyncReport_noReconcileAligned(t *testing.T) {
	tl := &artifacts.Timeline{
		Shots: []artifacts.TimelineShot{
			{AudioDurationSec: 10, DurationSec: 10.05},
			{AudioDurationSec: 12, DurationSec: 12.05},
		},
	}
	report := buildSyncReport(tl, 22, "", 0.05)
	if report.MaxDriftSec > syncDriftToleranceSec {
		t.Fatalf("expected aligned without pad, drift=%.3f", report.MaxDriftSec)
	}
	if !report.Aligned {
		t.Fatal("expected Aligned=true")
	}
}

func TestBuildSyncReport_reconcileWithoutPadFails(t *testing.T) {
	// 78s speech slots stretched to 120s video, narration file not padded
	tl := &artifacts.Timeline{
		Shots: []artifacts.TimelineShot{
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.16},
		},
	}
	report := buildSyncReport(tl, 77, "", 0.05)
	if report.Aligned {
		t.Fatalf("expected misaligned without pad, drift=%.3f", report.MaxDriftSec)
	}
	if report.MaxDriftSec < 40 {
		t.Fatalf("expected large compose drift, got %.3f", report.MaxDriftSec)
	}
}

func TestBuildSyncReport_reconcileWithPadAligned(t *testing.T) {
	if !ffmpeg.Available() {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	narrPath := filepath.Join(dir, "narration.mp3")
	if err := ffmpeg.GenerateSilentMP3(narrPath, 78); err != nil {
		t.Fatal(err)
	}
	tl := &artifacts.Timeline{
		Shots: []artifacts.TimelineShot{
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.14},
			{AudioDurationSec: 11, DurationSec: 17.16},
		},
	}
	if err := ffmpeg.PadAudioToDuration(narrPath, tl.TotalVideoSec()); err != nil {
		t.Fatal(err)
	}
	report := buildSyncReport(tl, 77, narrPath, 0.05)
	if report.SpeechAudioSec != 77 {
		t.Fatalf("expected speech_audio_sec=77, got %.2f", report.SpeechAudioSec)
	}
	if report.AudioTotalSec < 115 || report.AudioTotalSec > 125 {
		t.Fatalf("expected padded audio ~120s, got %.2f", report.AudioTotalSec)
	}
	if report.MaxDriftSec > syncDriftToleranceSec {
		t.Fatalf("expected aligned after pad, drift=%.3f video=%.2f audio=%.2f",
			report.MaxDriftSec, report.VideoTotalSec, report.AudioTotalSec)
	}
	if !report.Aligned {
		t.Fatal("expected Aligned=true after pad")
	}
}

func TestRefreshSyncReport_savesArtifact(t *testing.T) {
	if !ffmpeg.Available() {
		t.Skip("ffmpeg not available")
	}
	dir := t.TempDir()
	rc := &runctx.Context{RunDir: dir}
	narrPath := rc.ArtifactPath("artifacts/narration.mp3")
	if err := os.MkdirAll(filepath.Dir(narrPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ffmpeg.GenerateSilentMP3(narrPath, 30); err != nil {
		t.Fatal(err)
	}
	tl := &artifacts.Timeline{
		Shots: []artifacts.TimelineShot{
			{AudioDurationSec: 15, DurationSec: 15.05},
			{AudioDurationSec: 15, DurationSec: 15.05},
		},
	}
	if err := ffmpeg.PadAudioToDuration(narrPath, tl.TotalVideoSec()); err != nil {
		t.Fatal(err)
	}
	report, err := refreshSyncReport(rc, tl, 0.05, 30)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Aligned {
		t.Fatalf("expected aligned, drift=%.3f", report.MaxDriftSec)
	}
	loaded, err := artifacts.LoadSyncReport(rc.ArtifactPath("artifacts/sync-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.MaxDriftSec != report.MaxDriftSec {
		t.Fatalf("saved drift mismatch: got %.3f want %.3f", loaded.MaxDriftSec, report.MaxDriftSec)
	}
}
