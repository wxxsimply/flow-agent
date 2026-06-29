package skills

import (
	"os"
	"path/filepath"
	"strings"
)

// Skill 项目内 Agent Skill（.cursor/skills/<name>/SKILL.md + references/）。
type Skill struct {
	Name        string
	Description string
	Body        string
	References  map[string]string // 相对 references/ 的文件名 -> 正文
}

// Reference 按文件名读取 reference（如 motion-principles.md）。
func (s *Skill) Reference(name string) string {
	if s == nil || s.References == nil {
		return ""
	}
	return s.References[name]
}

func loadSkill(dir string) (*Skill, error) {
	skillPath := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return nil, err
	}
	name, desc, body := parseFrontmatter(string(data))
	if name == "" {
		name = filepath.Base(dir)
	}
	sk := &Skill{Name: name, Description: desc, Body: strings.TrimSpace(body), References: map[string]string{}}
	refDir := filepath.Join(dir, "references")
	entries, err := os.ReadDir(refDir)
	if err != nil {
		return sk, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
			continue
		}
		b, readErr := os.ReadFile(filepath.Join(refDir, e.Name()))
		if readErr != nil {
			continue
		}
		sk.References[e.Name()] = strings.TrimSpace(string(b))
	}
	return sk, nil
}

func parseFrontmatter(raw string) (name, desc, body string) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return "", "", raw
	}
	rest := strings.TrimPrefix(raw, "---")
	rest = strings.TrimSpace(rest)
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", raw
	}
	fm := rest[:end]
	body = strings.TrimSpace(rest[end+4:])
	for _, line := range strings.Split(fm, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "description:") {
			desc = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return name, desc, body
}
