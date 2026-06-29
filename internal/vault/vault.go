// Package vault 管理系列级设定（bible）与集摘要归档（文件型 MVP）。
package vault

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/config"
)

// SeriesVault 对应 series/<series_id>/ 目录。
type SeriesVault struct {
	SeriesID string
	Dir      string
}

// ForSeries 构造指定系列的 Vault 访问器。
func ForSeries(app *config.App, seriesID string) *SeriesVault {
	return &SeriesVault{
		SeriesID: seriesID,
		Dir:      filepath.Join(app.SeriesDir, seriesID),
	}
}

// Ensure 创建 vault 子目录。
func (v *SeriesVault) Ensure() error {
	return os.MkdirAll(filepath.Join(v.Dir, "vault"), 0o755)
}

// BiblePath 系列 bible 文件路径。
func (v *SeriesVault) BiblePath() string {
	return filepath.Join(v.Dir, "series-bible.yaml")
}

// LoadPreviousEpisodeSummary 读取上一集 vault 摘要；不存在或首集时返回空串。
func (v *SeriesVault) LoadPreviousEpisodeSummary(episodeNo int) string {
	if episodeNo <= 1 {
		return ""
	}
	path := filepath.Join(v.Dir, "vault", fmt.Sprintf("episode-%03d-summary.md", episodeNo-1))
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// LoadBible 读取 bible；不存在时返回占位文本。
func (v *SeriesVault) LoadBible() (string, error) {
	data, err := os.ReadFile(v.BiblePath())
	if err != nil {
		return fmt.Sprintf("series_id: %s\n(title: demo series)", v.SeriesID), nil
	}
	return string(data), nil
}

// ArchiveEpisode 将本集摘要写入 vault/episode-NNN-summary.md。
func (v *SeriesVault) ArchiveEpisode(episodeNo int, runID string) error {
	if err := v.Ensure(); err != nil {
		return err
	}
	summaryPath := filepath.Join(v.Dir, "vault", fmt.Sprintf("episode-%03d-summary.md", episodeNo))
	summary := fmt.Sprintf("# Episode %d summary\n\nArchived from run %s.\n", episodeNo, runID)
	return os.WriteFile(summaryPath, []byte(summary), 0o644)
}

// Search 全文检索 vault（FTS5）；query 为空时返回错误。
func (v *SeriesVault) Search(query string) ([]SearchHit, error) {
	return v.SearchFTS(query, 20)
}
