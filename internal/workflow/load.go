package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load 从 dir 读取 name.yaml 并解析为 Definition。
func Load(dir, name string) (*Definition, error) {
	path := filepath.Join(dir, name+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load workflow %q: %w", name, err)
	}
	var def Definition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse workflow %q: %w", name, err)
	}
	if def.Name == "" {
		def.Name = name // 文件内未写 name 时用文件名
	}
	return &def, nil
}
