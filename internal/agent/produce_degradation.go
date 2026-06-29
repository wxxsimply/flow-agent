package agent

import (
	"encoding/json"
	"os"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func flushProduceDegradation(rc *runctx.Context, sb *artifacts.Storyboard, tl *artifacts.Timeline) error {
	if rc == nil || sb == nil || tl == nil {
		return nil
	}
	plannedAI := artifacts.CountStoryboardVisualType(sb.Shots, "ai_video")
	actualKB := artifacts.CountVisualTypeShots(tl.Shots, "ken_burns")
	plannedKB := artifacts.CountStoryboardVisualType(sb.Shots, "ken_burns")

	timingMu.Lock()
	var kbShots []string
	var errSamples []string
	degraded := 0
	for _, t := range timings {
		if t.Degraded || t.Error == ErrUseKenBurns.Error() {
			degraded++
			kbShots = append(kbShots, t.ShotID)
		}
		if t.Error != "" && t.Error != ErrUseKenBurns.Error() {
			if len(errSamples) < 4 {
				errSamples = append(errSamples, t.ShotID+": "+t.Error)
			}
		}
	}
	timingMu.Unlock()

	bonScoreMu.Lock()
	bonEntries := bonScoresLocked()
	bonScoreMu.Unlock()

	report := &artifacts.ProduceDegradationReport{
		KenBurnsShots:   kbShots,
		DegradedCount:   degraded,
		PlannedAIVideo:  plannedAI,
		ActualKenBurns:  actualKB,
		APIErrorSamples: errSamples,
		WMRewardBoN:     bonEntries,
	}
	if actualKB > plannedKB {
		report.DegradedCount = actualKB - plannedKB
	}
	path := rc.ArtifactPath("artifacts/produce-degradation.json")
	if err := report.Save(path); err != nil {
		return err
	}
	rc.RecordArtifact("produce-degradation.json", "artifacts/produce-degradation.json", false)
	return nil
}

func loadProduceTimings(path string) ([]ProduceTimingEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []ProduceTimingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
