package vault

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// legacyInferredSecretMarker 旧版本 inferCharacterPatch 硬编码注入的污染标记。
// 任何 known_secrets 条目包含该子串都视为旧 bug 残留，自动清理。
const legacyInferredSecretMarker = "顾沉为保护林晚而假装分手的真相（录音已揭示）"

type bibleFile struct {
	Characters []struct {
		Name   string `yaml:"name"`
		Role   string `yaml:"role"`
		Traits string `yaml:"traits"`
	} `yaml:"characters"`
}

// EnsureCharacterStateFromBible 处理两件事：
//  1. 自动清洗历史 inferCharacterPatch 写入的污染条目（known_secrets 中的硬编码字符串）。
//  2. 若清洗后 state 仍为空，从 series-bible 初始化角色基线。
func (v *SeriesVault) EnsureCharacterStateFromBible() error {
	state, err := v.LoadCharacterState()
	if err != nil {
		return err
	}

	if cleaned := cleanLegacyInferredPatch(state); cleaned {
		fmt.Fprintln(os.Stderr, "vault: cleaned legacy inferred character patch")
		if err := v.writeCharacterState(state); err != nil {
			return err
		}
	}

	if len(state) > 0 {
		return nil
	}

	data, err := os.ReadFile(v.BiblePath())
	if err != nil {
		return nil
	}
	var bf bibleFile
	if err := yaml.Unmarshal(data, &bf); err != nil {
		return nil
	}
	chars := map[string]any{}
	for _, c := range bf.Characters {
		if c.Name == "" {
			continue
		}
		chars[c.Name] = map[string]any{
			"role":          c.Role,
			"traits":        c.Traits,
			"known_secrets": []any{},
		}
	}
	if len(chars) == 0 {
		return nil
	}
	return v.ApplyCharacterPatch(chars)
}

// cleanLegacyInferredPatch 删除所有 known_secrets 中匹配 legacyInferredSecretMarker 的条目，
// 并移除随之注入的 notes 字段。返回 true 表示发生了修改。
func cleanLegacyInferredPatch(state map[string]any) bool {
	changed := false
	for name, raw := range state {
		obj, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		secretsAny, has := obj["known_secrets"]
		if !has {
			continue
		}
		secrets, ok := secretsAny.([]any)
		if !ok {
			continue
		}
		filtered := make([]any, 0, len(secrets))
		removedAny := false
		for _, item := range secrets {
			if s, ok := item.(string); ok && strings.Contains(s, legacyInferredSecretMarker) {
				removedAny = true
				continue
			}
			filtered = append(filtered, item)
		}
		if !removedAny {
			continue
		}
		obj["known_secrets"] = filtered
		if notes, ok := obj["notes"].(string); ok && strings.Contains(notes, "听完录音后情绪克制") {
			delete(obj, "notes")
		}
		state[name] = obj
		changed = true
	}
	return changed
}
