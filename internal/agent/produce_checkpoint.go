package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/compose/ffmpeg"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

// ProduceCheckpoint 镜级 produce 进度（断点续产）。
type ProduceCheckpoint struct {
	CompletedShots []string          `json:"completed_shots"`
	Errors         map[string]string `json:"errors,omitempty"`
}

func loadProduceCheckpoint(rc *runctx.Context) *ProduceCheckpoint {
	if rc == nil {
		return &ProduceCheckpoint{Errors: map[string]string{}}
	}
	path := rc.ArtifactPath("artifacts/produce-checkpoint.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return &ProduceCheckpoint{Errors: map[string]string{}}
	}
	var cp ProduceCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return &ProduceCheckpoint{Errors: map[string]string{}}
	}
	if cp.Errors == nil {
		cp.Errors = map[string]string{}
	}
	return &cp
}

// LoadProduceCheckpointForResume 供 resume CLI 读取镜级进度。
func LoadProduceCheckpointForResume(rc *runctx.Context) *ProduceCheckpoint {
	return loadProduceCheckpoint(rc)
}

func saveProduceCheckpoint(rc *runctx.Context, cp *ProduceCheckpoint) {
	if rc == nil || cp == nil {
		return
	}
	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return
	}
	_ = rc.WriteArtifact("artifacts/produce-checkpoint.json", data)
	rc.RecordArtifact("produce-checkpoint.json", "artifacts/produce-checkpoint.json", false)
}

func shotClipExists(vidPath string) bool {
	if vidPath == "" {
		return false
	}
	if _, err := os.Stat(vidPath); err != nil {
		return false
	}
	dur, err := ffmpeg.ProbeVideoDurationSec(vidPath)
	return err == nil && dur > 0.05
}

func markShotCheckpoint(rc *runctx.Context, shotID, vidPath string, produceErr error) {
	cp := loadProduceCheckpoint(rc)
	found := false
	for _, id := range cp.CompletedShots {
		if id == shotID {
			found = true
			break
		}
	}
	if produceErr == nil && shotClipExists(vidPath) {
		if !found {
			cp.CompletedShots = append(cp.CompletedShots, shotID)
		}
		delete(cp.Errors, shotID)
	} else if produceErr != nil {
		cp.Errors[shotID] = produceErr.Error()
	}
	saveProduceCheckpoint(rc, cp)
}

func shouldSkipExistingShot(vidPath string) bool {
	return shotClipExists(vidPath)
}

func logResumeSkippedShots(rc *runctx.Context, assetsDir string, shotIDs []string) {
	if rc == nil {
		return
	}
	n := 0
	for _, id := range shotIDs {
		if shouldSkipExistingShot(filepath.Join(assetsDir, id+".mp4")) {
			n++
		}
	}
	if n > 0 {
		// logged at produce start
		_ = n
	}
}
