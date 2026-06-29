package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// CharacterStatePath 系列角色状态文件路径（位于 vault/ 子目录）。
func (v *SeriesVault) CharacterStatePath() string {
	return filepath.Join(v.Dir, "vault", "character-state.json")
}

// legacyCharacterStatePath 旧版本路径，用于一次性迁移。
func (v *SeriesVault) legacyCharacterStatePath() string {
	return filepath.Join(v.Dir, "character-state.json")
}

// migrateLegacyCharacterState 若旧路径存在且新路径不存在，迁移至 vault/ 下。
func (v *SeriesVault) migrateLegacyCharacterState() error {
	newPath := v.CharacterStatePath()
	if _, err := os.Stat(newPath); err == nil {
		return nil
	}
	oldPath := v.legacyCharacterStatePath()
	if _, err := os.Stat(oldPath); err != nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

// LoadCharacterState 读取角色状态；不存在时返回空 map。
func (v *SeriesVault) LoadCharacterState() (map[string]any, error) {
	_ = v.migrateLegacyCharacterState()
	path := v.CharacterStatePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ApplyCharacterPatch 将 patch 合并进 character-state.json（浅合并顶层键）。
func (v *SeriesVault) ApplyCharacterPatch(patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	state, err := v.LoadCharacterState()
	if err != nil {
		return err
	}
	for k, val := range patch {
		state[k] = val
	}
	return v.writeCharacterState(state)
}

// writeCharacterState 序列化并写入 vault/character-state.json。
func (v *SeriesVault) writeCharacterState(state map[string]any) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	path := v.CharacterStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// ApplyCharacterPatchFile 从 JSON 文件读取 patch 并应用。
func (v *SeriesVault) ApplyCharacterPatchFile(patchPath string) error {
	data, err := os.ReadFile(patchPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var patch map[string]any
	if err := json.Unmarshal(data, &patch); err != nil {
		return fmt.Errorf("character patch json: %w", err)
	}
	return v.ApplyCharacterPatch(patch)
}
