package vault

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	docKindBible           = "bible"
	docKindEpisodeSummary  = "episode_summary"
	docKindForeshadow      = "foreshadow"
)

// SearchHit FTS 检索命中。
type SearchHit struct {
	Kind      string
	EpisodeNo int
	Title     string
	Snippet   string
	Score     float64
}

// openIndexDB 打开或创建 series/<id>/vault/index.db。
func (v *SeriesVault) openIndexDB() (*sql.DB, error) {
	if err := v.Ensure(); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(v.Dir, "vault", "index.db")
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if err := migrateIndex(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func migrateIndex(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			kind TEXT NOT NULL,
			episode_no INTEGER NOT NULL DEFAULT 0,
			title TEXT,
			body TEXT NOT NULL,
			source_path TEXT,
			updated_at TEXT NOT NULL,
			UNIQUE(kind, episode_no, source_path)
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS documents_fts USING fts5(
			title, body,
			content='documents', content_rowid='id',
			tokenize='unicode61'
		)`,
		`CREATE TRIGGER IF NOT EXISTS documents_ai AFTER INSERT ON documents BEGIN
			INSERT INTO documents_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
		END`,
		`CREATE TRIGGER IF NOT EXISTS documents_ad AFTER DELETE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, title, body) VALUES('delete', old.id, old.title, old.body);
		END`,
		`CREATE TRIGGER IF NOT EXISTS documents_au AFTER UPDATE ON documents BEGIN
			INSERT INTO documents_fts(documents_fts, rowid, title, body) VALUES('delete', old.id, old.title, old.body);
			INSERT INTO documents_fts(rowid, title, body) VALUES (new.id, new.title, new.body);
		END`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// SyncIndex 将 bible、集摘要、伏笔表写入 FTS 索引。
func (v *SeriesVault) SyncIndex() error {
	db, err := v.openIndexDB()
	if err != nil {
		return err
	}
	defer db.Close()

	bible, err := v.LoadBible()
	if err != nil {
		return err
	}
	if err := upsertDocument(db, docKindBible, 0, "series-bible", v.BiblePath(), bible); err != nil {
		return err
	}

	vaultDir := filepath.Join(v.Dir, "vault")
	entries, _ := os.ReadDir(vaultDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "episode-") || !strings.HasSuffix(e.Name(), "-summary.md") {
			continue
		}
		path := filepath.Join(vaultDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		ep := parseEpisodeNoFromSummaryName(e.Name())
		title := fmt.Sprintf("Episode %d summary", ep)
		if err := upsertDocument(db, docKindEpisodeSummary, ep, title, path, string(data)); err != nil {
			return err
		}
	}

	foreshadowPath := filepath.Join(v.Dir, "foreshadows.yaml")
	if data, err := os.ReadFile(foreshadowPath); err == nil {
		if err := upsertDocument(db, docKindForeshadow, 0, "foreshadows", foreshadowPath, string(data)); err != nil {
			return err
		}
	}
	return nil
}

func parseEpisodeNoFromSummaryName(name string) int {
	var ep int
	_, _ = fmt.Sscanf(name, "episode-%d-summary.md", &ep)
	return ep
}

func upsertDocument(db *sql.DB, kind string, episodeNo int, title, sourcePath, body string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO documents (kind, episode_no, title, body, source_path, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(kind, episode_no, source_path) DO UPDATE SET
			title=excluded.title, body=excluded.body, updated_at=excluded.updated_at`,
		kind, episodeNo, title, body, sourcePath, now,
	)
	return err
}

// IndexEpisodeSummary 写入摘要文件并更新索引。
func (v *SeriesVault) IndexEpisodeSummary(episodeNo int, summary string) error {
	if err := v.Ensure(); err != nil {
		return err
	}
	path := filepath.Join(v.Dir, "vault", fmt.Sprintf("episode-%03d-summary.md", episodeNo))
	if err := os.WriteFile(path, []byte(summary), 0o644); err != nil {
		return err
	}
	db, err := v.openIndexDB()
	if err != nil {
		return err
	}
	defer db.Close()
	title := fmt.Sprintf("Episode %d summary", episodeNo)
	return upsertDocument(db, docKindEpisodeSummary, episodeNo, title, path, summary)
}

// SearchFTS 全文检索 vault；FTS 无结果时回退 LIKE 搜索（兼容中文分词不佳的情况）。
func (v *SeriesVault) SearchFTS(query string, limit int) ([]SearchHit, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if err := v.SyncIndex(); err != nil {
		return nil, err
	}
	db, err := v.openIndexDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	if limit <= 0 {
		limit = 20
	}

	hits, err := searchFTS5(db, query, limit)
	if err != nil || len(hits) == 0 {
		likeHits, likeErr := searchLIKE(db, query, limit)
		if likeErr == nil && len(likeHits) > 0 {
			return likeHits, nil
		}
	}
	return hits, err
}

func searchFTS5(db *sql.DB, query string, limit int) ([]SearchHit, error) {
	ftsQuery := buildFTSQuery(query)
	rows, err := db.Query(`
		SELECT d.kind, d.episode_no, d.title,
			snippet(documents_fts, 1, '**', '**', '…', 32) AS snip,
			bm25(documents_fts) AS score
		FROM documents_fts
		JOIN documents d ON d.id = documents_fts.rowid
		WHERE documents_fts MATCH ?
		ORDER BY score
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

func searchLIKE(db *sql.DB, query string, limit int) ([]SearchHit, error) {
	pattern := "%" + query + "%"
	rows, err := db.Query(`
		SELECT kind, episode_no, title,
			substr(body, max(1, instr(body, ?1) - 30), 80) AS snip,
			0.0 AS score
		FROM documents
		WHERE body LIKE ?2 OR title LIKE ?2
		LIMIT ?3`, query, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHits(rows)
}

func scanHits(rows *sql.Rows) ([]SearchHit, error) {
	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.Kind, &h.EpisodeNo, &h.Title, &h.Snippet, &h.Score); err != nil {
			return nil, err
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// buildFTSQuery 将用户输入转为 FTS5 MATCH 表达式（每词加引号）。
func buildFTSQuery(q string) string {
	q = strings.TrimSpace(q)
	parts := strings.Fields(q)
	if len(parts) == 0 {
		return `""`
	}
	for i, p := range parts {
		p = strings.ReplaceAll(p, `"`, "")
		parts[i] = `"` + p + `"`
	}
	return strings.Join(parts, " OR ")
}
