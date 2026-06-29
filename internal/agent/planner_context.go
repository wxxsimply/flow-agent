package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/internal/vault"
)

// plannerInjectContext 上集摘要、发布指标与下集 hints（阶段 I）。
func plannerInjectContext(v *vault.SeriesVault, episodeNo int) (prevSummary, publishMetrics, nextHints string) {
	prevSummary = v.LoadPreviousEpisodeSummary(episodeNo)
	if m, err := v.LoadPreviousPublishMetrics(episodeNo); err == nil && m != nil {
		publishMetrics = m.FormatForPlanner()
	}
	nextHints = strings.TrimSpace(v.LoadNextEpisodeHints(episodeNo))
	return prevSummary, publishMetrics, nextHints
}
