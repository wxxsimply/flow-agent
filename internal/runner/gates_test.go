package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func TestCheckLengthInRange(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{
		RunDir: dir,
		Manifest: &artifacts.Manifest{Gates: map[string]bool{}},
	}
	var s string
	for i := 0; i < 800; i++ {
		s += "字"
	}
	path := filepath.Join(dir, "artifacts", "chapter.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkLengthInRange(rc); err != nil {
		t.Fatal(err)
	}

	short := "太短"
	if err := os.WriteFile(path, []byte(short), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkLengthInRange(rc); err == nil {
		t.Fatal("expected error for short chapter")
	}
}

func TestCheckAVSyncOK(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{RunDir: dir, Manifest: &artifacts.Manifest{Gates: map[string]bool{}}}
	path := filepath.Join(dir, "artifacts", "sync-report.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	ok := `{"video_total_sec":180,"audio_total_sec":179.5,"max_drift_sec":0.2,"aligned":true}`
	if err := os.WriteFile(path, []byte(ok), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkAVSyncOK(rc); err != nil {
		t.Fatal(err)
	}
	bad := `{"max_drift_sec":1.2,"aligned":false}`
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkAVSyncOK(rc); err == nil {
		t.Fatal("expected av_sync_ok failure")
	}
}

func TestCheckDurationOK(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{
		RunDir:    dir,
		Manifest:  &artifacts.Manifest{Gates: map[string]bool{}},
		EpisodeNo: 1,
	}
	rc.Def = nil

	sb := `{
  "episode_no": 1,
  "target_duration_sec": 180,
  "total_narration_sec": 180,
  "shots": [
    {"id":"s01","duration_sec":37.5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"a","subtitle":"a"},
    {"id":"s02","duration_sec":37.5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"b","subtitle":"b"},
    {"id":"s03","duration_sec":37.5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"c","subtitle":"c"},
    {"id":"s04","duration_sec":37.5,"visual_type":"ai_video","ai_video_budget":true,"visual_prompt":"x","narration":"d","subtitle":"d"}
  ]
}`
	path := filepath.Join(dir, "artifacts", "storyboard.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(sb), 0o644); err != nil {
		t.Fatal(err)
	}

	// TargetDurationSec defaults to 150 when Def is nil
	if err := checkDurationOK(rc); err != nil {
		t.Fatal(err)
	}

	bad := strings.Replace(sb, `"duration_sec":37.5`, `"duration_sec":10`, 4)
	if err := os.WriteFile(path, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkDurationOK(rc); err == nil {
		t.Fatal("expected duration_ok failure")
	}
}

func TestCheckMotionQualityOK(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{RunDir: dir, Manifest: &artifacts.Manifest{Gates: map[string]bool{}}}
	artDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sb := `{
  "shots": [
    {"id":"s01","visual_type":"ai_video","duration_sec":5,"narration":"a","visual_prompt":"x"},
    {"id":"s02","visual_type":"ai_video","duration_sec":5,"narration":"b","visual_prompt":"x"}
  ]
}`
	tl := `{
  "shots": [
    {"id":"s01","visual_type":"ken_burns","duration_sec":5},
    {"id":"s02","visual_type":"ken_burns","duration_sec":5}
  ]
}`
	if err := os.WriteFile(filepath.Join(artDir, "storyboard.json"), []byte(sb), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "timeline.json"), []byte(tl), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkMotionQualityOK(rc); err == nil {
		t.Fatal("expected motion_quality_ok failure for unplanned ken_burns")
	}
}

func TestCheckAudioDurationOK(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{RunDir: dir, Manifest: &artifacts.Manifest{Gates: map[string]bool{}}, Workflow: "micro-movie"}
	artDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	syncOK := `{"audio_total_sec":23.0,"video_total_sec":23.0,"max_drift_sec":0.1,"aligned":true}`
	sb := `{"total_narration_sec":23.0,"shots":[{"id":"s01","duration_sec":23,"narration":"test","visual_prompt":"x","visual_type":"ai_video"}]}`
	if err := os.WriteFile(filepath.Join(artDir, "sync-report.json"), []byte(syncOK), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artDir, "storyboard.json"), []byte(sb), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkAudioDurationOK(rc); err != nil {
		t.Fatal(err)
	}
	badSync := `{"audio_total_sec":5.0,"video_total_sec":5.0,"max_drift_sec":0.1,"aligned":true}`
	if err := os.WriteFile(filepath.Join(artDir, "sync-report.json"), []byte(badSync), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkAudioDurationOK(rc); err == nil {
		t.Fatal("expected audio_duration_ok failure")
	}
}

func TestCheckAudioDurationOK_paddedFileVsSpeech(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{
		RunDir:   dir,
		Workflow: "micro-movie",
		Creative: &artifacts.CreativeOptions{InputMode: "director"},
		Manifest: &artifacts.Manifest{Gates: map[string]bool{}},
	}
	artDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sb := `{"total_narration_sec":77.1,"shots":[{"id":"s01","duration_sec":11,"narration":"test","visual_prompt":"x","visual_type":"ai_video"}]}`
	if err := os.WriteFile(filepath.Join(artDir, "storyboard.json"), []byte(sb), 0o644); err != nil {
		t.Fatal(err)
	}
	paddedSync := `{"speech_audio_sec":77.1,"audio_total_sec":120.0,"video_total_sec":120.0,"max_drift_sec":0.02,"aligned":true}`
	if err := os.WriteFile(filepath.Join(artDir, "sync-report.json"), []byte(paddedSync), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkAudioDurationOK(rc); err != nil {
		t.Fatalf("expected pass with speech_audio_sec vs storyboard: %v", err)
	}
	badSpeech := `{"speech_audio_sec":5.0,"audio_total_sec":120.0,"video_total_sec":120.0,"max_drift_sec":0.02,"aligned":true}`
	if err := os.WriteFile(filepath.Join(artDir, "sync-report.json"), []byte(badSpeech), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkAudioDurationOK(rc); err == nil {
		t.Fatal("expected audio_duration_ok failure when speech diverges from storyboard")
	}
}

func TestCheckNarrationCompleteOK(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{RunDir: dir}
	artDir := filepath.Join(dir, "artifacts")
	if err := os.MkdirAll(artDir, 0o755); err != nil {
		t.Fatal(err)
	}
	okSB := `{"shots":[{"id":"s01","narration":"完整旁白。"}]}`
	if err := os.WriteFile(filepath.Join(artDir, "storyboard.json"), []byte(okSB), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkNarrationCompleteOK(rc); err != nil {
		t.Fatalf("expected pass: %v", err)
	}
	badSB := `{"shots":[{"id":"s01","narration":"截断旁白，"}]}`
	if err := os.WriteFile(filepath.Join(artDir, "storyboard.json"), []byte(badSB), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkNarrationCompleteOK(rc); err == nil {
		t.Fatal("expected narration_complete_ok failure")
	}
}

func TestCheckNoBlockIssues(t *testing.T) {
	dir := t.TempDir()
	rc := &runctx.Context{RunDir: dir}

	clean := `{"episode_no":1,"blocked":false,"block_count":0,"warning_count":0,"blocks":[],"warnings":[],"checked_at":"test"}`
	path := filepath.Join(dir, "artifacts", "compliance-report.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(clean), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkNoBlockIssues(rc); err != nil {
		t.Fatal(err)
	}

	blocked := `{"episode_no":1,"blocked":true,"block_count":1,"blocks":[{"severity":"block","word":"赌博","source":"chapter.md"}],"warnings":[]}`
	if err := os.WriteFile(path, []byte(blocked), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkNoBlockIssues(rc); err == nil {
		t.Fatal("expected no_block_issues failure")
	}
}

func TestCheckGates_defersBriefConfirmedOnStopAfterAssemble(t *testing.T) {
	rc := &runctx.Context{
		StopAfterStage: "assemble",
		AutoGate:       false,
		Manifest:       &artifacts.Manifest{Gates: map[string]bool{}},
	}
	stage := &workflow.StageDefinition{
		ID: "assemble",
		Gates: []workflow.GateDefinition{
			{ID: "brief_confirmed", Type: "human", Skippable: true},
		},
	}
	if err := CheckGates(rc, stage); err != nil {
		t.Fatalf("expected defer brief_confirmed: %v", err)
	}
}
