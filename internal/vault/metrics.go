package vault

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// PublishMetricsDir 系列级发布指标目录。
func (v *SeriesVault) PublishMetricsDir() string {
	return filepath.Join(v.Dir, "publish-metrics")
}

// PublishMetricsPath 单集指标文件路径。
func (v *SeriesVault) PublishMetricsPath(episodeNo int) string {
	return filepath.Join(v.PublishMetricsDir(), fmt.Sprintf("episode-%03d.json", episodeNo))
}

// NextHintsPath 面向指定集数的下集策划提示（由上一集 learn 写入）。
func (v *SeriesVault) NextHintsPath(forEpisodeNo int) string {
	return filepath.Join(v.Dir, "vault", fmt.Sprintf("episode-%03d-next-hints.md", forEpisodeNo))
}

// SavePublishMetrics 写入 series/publish-metrics/episode-NNN.json。
func (v *SeriesVault) SavePublishMetrics(m *artifacts.PublishMetrics) error {
	if err := os.MkdirAll(v.PublishMetricsDir(), 0o755); err != nil {
		return err
	}
	m.SeriesID = v.SeriesID
	return m.Save(v.PublishMetricsPath(m.EpisodeNo))
}

// LoadPublishMetrics 读取指定集发布指标；不存在返回 nil, nil。
func (v *SeriesVault) LoadPublishMetrics(episodeNo int) (*artifacts.PublishMetrics, error) {
	path := v.PublishMetricsPath(episodeNo)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}
	return artifacts.LoadPublishMetrics(path)
}

// LoadPreviousPublishMetrics 读取上一集（episodeNo-1）指标。
func (v *SeriesVault) LoadPreviousPublishMetrics(episodeNo int) (*artifacts.PublishMetrics, error) {
	if episodeNo <= 1 {
		return nil, nil
	}
	return v.LoadPublishMetrics(episodeNo - 1)
}

// SaveNextEpisodeHints 保存面向 forEpisodeNo 的下集提示。
func (v *SeriesVault) SaveNextEpisodeHints(forEpisodeNo int, markdown string) error {
	if err := v.Ensure(); err != nil {
		return err
	}
	return os.WriteFile(v.NextHintsPath(forEpisodeNo), []byte(markdown), 0o644)
}

// LoadNextEpisodeHints 读取面向当前集数的策划提示；不存在返回空串。
func (v *SeriesVault) LoadNextEpisodeHints(episodeNo int) string {
	data, err := os.ReadFile(v.NextHintsPath(episodeNo))
	if err != nil {
		return ""
	}
	return string(data)
}

// ListPublishMetricsEpisodes 列出已有指标文件的集数（升序）。
func (v *SeriesVault) ListPublishMetricsEpisodes() ([]int, error) {
	dir := v.PublishMetricsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var eps []int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".json")
		if !strings.HasPrefix(base, "episode-") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(base, "episode-"))
		if err != nil || n <= 0 {
			continue
		}
		eps = append(eps, n)
	}
	sort.Ints(eps)
	return eps, nil
}
