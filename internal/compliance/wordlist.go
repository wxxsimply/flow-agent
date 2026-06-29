package compliance

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WordEntry 词库条目。
type WordEntry struct {
	Term     string
	Severity string // block | warning
	Source   string // 文件名，便于调试
}

// LoadWordLists 从 config/compliance/ 加载自定义词库与平台词库。
func LoadWordLists(configDir string) ([]WordEntry, error) {
	files := []string{
		filepath.Join(configDir, "compliance", "words.txt"),
		filepath.Join(configDir, "compliance", "platform-douyin.txt"),
	}
	var all []WordEntry
	for _, path := range files {
		entries, err := loadWordFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		all = append(all, entries...)
	}
	return dedupeEntries(all), nil
}

func loadWordFile(path string) ([]WordEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	base := filepath.Base(path)
	var out []WordEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sev := "block"
		term := line
		if strings.HasPrefix(line, "warn:") {
			sev = "warning"
			term = strings.TrimSpace(strings.TrimPrefix(line, "warn:"))
		} else if strings.HasPrefix(line, "block:") {
			term = strings.TrimSpace(strings.TrimPrefix(line, "block:"))
		}
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		out = append(out, WordEntry{Term: term, Severity: sev, Source: base})
	}
	return out, sc.Err()
}

func dedupeEntries(in []WordEntry) []WordEntry {
	type key struct {
		term string
		sev  string
	}
	seen := map[key]WordEntry{}
	for _, e := range in {
		k := key{term: strings.ToLower(e.Term), sev: e.Severity}
		if _, ok := seen[k]; !ok {
			seen[k] = e
		}
	}
	out := make([]WordEntry, 0, len(seen))
	for _, e := range seen {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].Term) != len(out[j].Term) {
			return len(out[i].Term) > len(out[j].Term)
		}
		return out[i].Term < out[j].Term
	})
	return out
}
