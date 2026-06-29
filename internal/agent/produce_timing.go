package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// ProduceTimingEntry 单镜 produce 耗时与策略。
type ProduceTimingEntry struct {
	ShotID        string `json:"shot_id"`
	Tier          string `json:"tier,omitempty"`
	MultiKeyframe bool   `json:"multi_keyframe"`
	BoNEnabled    bool   `json:"bon_enabled"`
	BoNCandidates int    `json:"bon_candidates,omitempty"`
	I2VModel      string `json:"i2v_model,omitempty"`
	WallMs        int64  `json:"wall_ms"`
	Error         string `json:"error,omitempty"`
	Degraded      bool   `json:"degraded,omitempty"`
	VisualType    string `json:"visual_type,omitempty"`
}

var (
	timingMu sync.Mutex
	timings  []ProduceTimingEntry
)

func resetProduceTimings() {
	timingMu.Lock()
	timings = nil
	timingMu.Unlock()
}

func recordProduceTiming(shot artifacts.Shot, pol ShotProducePolicy, d time.Duration, err error) {
	e := ProduceTimingEntry{
		ShotID:        shot.ID,
		Tier:          shot.Tier,
		MultiKeyframe: pol.MultiKeyframe,
		BoNEnabled:    pol.BoNEnabled,
		BoNCandidates: pol.BoNCandidates,
		I2VModel:      pol.I2VModel,
		WallMs:        d.Milliseconds(),
	}
	if err != nil {
		e.Error = err.Error()
		if errors.Is(err, ErrUseKenBurns) {
			e.Degraded = true
			e.VisualType = "ken_burns"
		}
	}
	timingMu.Lock()
	timings = append(timings, e)
	timingMu.Unlock()
}

func flushProduceTimings(rc *runctx.Context) error {
	timingMu.Lock()
	data, err := json.MarshalIndent(timings, "", "  ")
	summary := produceTimingSummaryLocked()
	timingMu.Unlock()
	if err != nil {
		return err
	}
	if len(data) == 0 {
		data = []byte("[]")
	}
	if err := rc.WriteArtifact("artifacts/produce-timing.json", data); err != nil {
		return err
	}
	rc.RecordArtifact("produce-timing.json", "artifacts/produce-timing.json", false)
	if summary != "" {
		_ = rc.WriteArtifact("artifacts/produce-timing-summary.txt", []byte(summary))
	}
	return nil
}

func produceTimingSummaryLocked() string {
	if len(timings) == 0 {
		return ""
	}
	var total int64
	bonJobs := 0
	for _, t := range timings {
		total += t.WallMs
		if t.BoNEnabled && t.BoNCandidates > 0 {
			bonJobs += t.BoNCandidates
		}
	}
	return fmt.Sprintf("shots=%d total_wall_ms=%d approx_bon_i2v_jobs=%d\n", len(timings), total, bonJobs)
}
