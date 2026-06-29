package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/vault"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunAnalyst 归档本集、合并发布指标、生成下集 hints 并写入 SeriesVault。
func RunAnalyst(rc *runctx.Context, v *vault.SeriesVault) error {
	metrics, err := resolveEpisodeMetrics(rc, v)
	if err != nil {
		return err
	}
	metrics.SeriesID = rc.SeriesID
	if err := metrics.Normalize(); err != nil {
		return err
	}

	snap, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	if err := rc.WriteArtifact("artifacts/metrics-snapshot.json", snap); err != nil {
		return err
	}
	if err := v.SavePublishMetrics(metrics); err != nil {
		return err
	}

	hints := buildNextEpisodeHints(rc, metrics)
	if err := rc.WriteArtifact("artifacts/next-episode-hints.md", []byte(hints)); err != nil {
		return err
	}
	if rc.EpisodeNo+1 > 0 {
		if err := v.SaveNextEpisodeHints(rc.EpisodeNo+1, hints); err != nil {
			return err
		}
	}

	summary := buildEpisodeSummary(rc)
	if err := v.IndexEpisodeSummary(rc.EpisodeNo, summary); err != nil {
		return err
	}

	rc.RecordArtifact("metrics-snapshot.json", "artifacts/metrics-snapshot.json", true)
	rc.RecordArtifact("next-episode-hints.md", "artifacts/next-episode-hints.md", true)
	return nil
}

func resolveEpisodeMetrics(rc *runctx.Context, v *vault.SeriesVault) (*artifacts.PublishMetrics, error) {
	if m, err := v.LoadPublishMetrics(rc.EpisodeNo); err != nil {
		return nil, err
	} else if m != nil && (m.Views24h > 0 || m.CompletionRate > 0 || len(m.CommentKeywords) > 0) {
		return m, nil
	}
	snapPath := rc.ArtifactPath("artifacts/metrics-snapshot.json")
	if data, err := os.ReadFile(snapPath); err == nil {
		var m artifacts.PublishMetrics
		if json.Unmarshal(data, &m) == nil && m.EpisodeNo > 0 {
			return &m, nil
		}
	}
	return &artifacts.PublishMetrics{
		EpisodeNo:      rc.EpisodeNo,
		Views24h:       0,
		CompletionRate: 0,
		Note:           "发布后请用 flowagent metrics set 填入真实数据",
	}, nil
}

func buildNextEpisodeHints(rc *runctx.Context, m *artifacts.PublishMetrics) string {
	base := BuildNextHintsFromMetrics(rc.EpisodeNo+1, m)
	var b strings.Builder
	b.WriteString(base)
	if !strings.HasSuffix(base, "\n") {
		b.WriteString("\n")
	}

	hookLine := hookLineFromPlan(rc)
	if hookLine != "" {
		fmt.Fprintf(&b, "\n## 上集钩子\n\n%s\n", hookLine)
		fmt.Fprintf(&b, "- 必须回应上集悬念：%s\n", hookLine)
	}

	brief := readArtifactSnippet(rc, "artifacts/episode-brief.md", 400)
	if brief != "" {
		b.WriteString("\n## 上集 brief 摘要\n\n")
		b.WriteString(brief)
		b.WriteString("\n")
	}
	return b.String()
}

func buildEpisodeSummary(rc *runctx.Context) string {
	chapterPath := rc.ArtifactPath("artifacts/chapter.md")
	data, err := os.ReadFile(chapterPath)
	if err != nil {
		return fmt.Sprintf("# Episode %d summary\n\nArchived from run %s.\n", rc.EpisodeNo, rc.RunID)
	}
	text := string(data)
	text = strings.TrimPrefix(text, "<!-- dry-run -->\n")
	if len([]rune(text)) > 600 {
		runes := []rune(text)
		text = string(runes[:600]) + "…"
	}
	return fmt.Sprintf("# Episode %d summary\n\nRun: %s\n\n%s\n", rc.EpisodeNo, rc.RunID, text)
}
