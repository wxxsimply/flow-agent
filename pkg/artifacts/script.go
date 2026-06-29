package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Script 微电影可拍剧本（Script 阶段产物）。
type Script struct {
	Title    string        `json:"title"`
	Logline  string        `json:"logline"`
	Scenes   []ScriptScene `json:"scenes"`
}

// ScriptScene 单场戏。
type ScriptScene struct {
	ID        int    `json:"id"`
	Heading   string `json:"heading"`
	Action    string `json:"action"`
	Narration string `json:"narration"`
}

// LoadScript 读取 script.json。
func LoadScript(path string) (*Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Script
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// ToMarkdown 转为分镜 LLM 可读的剧本文本。
func (s *Script) ToMarkdown() string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	if s.Title != "" {
		b.WriteString("# ")
		b.WriteString(s.Title)
		b.WriteString("\n\n")
	}
	if s.Logline != "" {
		b.WriteString("> ")
		b.WriteString(s.Logline)
		b.WriteString("\n\n")
	}
	for _, sc := range s.Scenes {
		fmt.Fprintf(&b, "## 场景 %d：%s\n\n", sc.ID, sc.Heading)
		if sc.Action != "" {
			b.WriteString(sc.Action)
			b.WriteString("\n\n")
		}
		if sc.Narration != "" {
			b.WriteString("**旁白：** ")
			b.WriteString(sc.Narration)
			b.WriteString("\n\n")
		}
	}
	return b.String()
}
