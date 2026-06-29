// Package workflow 解析 docs/workflows/*.yaml 工作流定义。
package workflow

// Definition 与 novel-short-douyin.yaml 等文件结构对应。
type Definition struct {
	Name          string            `yaml:"name"`
	Version       string            `yaml:"version"`
	Description   string            `yaml:"description"`
	Domain        string            `yaml:"domain"`
	Context       map[string]any    `yaml:"context"`       // 如 target_duration_sec
	Budget        map[string]any    `yaml:"budget"`        // token、成本上限
	Stream        map[string]any    `yaml:"stream"`        // 流式写作参数
	Stages        []StageDefinition `yaml:"stages"`        // 有序阶段列表
	Plugins       []Plugin          `yaml:"plugins"`
	Observability Observability     `yaml:"observability"`
}

// StageDefinition 单个流水线阶段（plan / write / …）。
type StageDefinition struct {
	ID        string         `yaml:"id"`
	Agent     string         `yaml:"agent"`      // 逻辑 Agent 名称
	DependsOn []string       `yaml:"depends_on"` // 依赖的前置阶段 id
	Mode      string         `yaml:"mode"`       // 如 stream
	Artifacts ArtifactRules  `yaml:"artifacts"`
	Gates     []GateDefinition `yaml:"gates"`
	Hooks     any            `yaml:"hooks"` // YAML 中可能是 map 或 list
	Sandbox   map[string]any `yaml:"sandbox"`
	Context   map[string]any `yaml:"context"`
}

// ArtifactRules 本阶段必须/可选产物路径。
type ArtifactRules struct {
	Required []ArtifactRule `yaml:"required"`
	Optional []ArtifactRule `yaml:"optional"`
}

// ArtifactRule 单个产物文件规则。
type ArtifactRule struct {
	Path   string `yaml:"path"`
	Schema string `yaml:"schema"` // 可选 JSON schema 名
}

// GateDefinition 人工或自动门禁。
type GateDefinition struct {
	ID        string `yaml:"id"`
	Type      string `yaml:"type"` // human | automatic
	Skippable bool   `yaml:"skippable"`
	Condition string `yaml:"condition"`
}

// Plugin 工作流引用的扩展插件（Provider 能力声明）。
type Plugin struct {
	Name     string   `yaml:"name"`
	Config   string   `yaml:"config"`
	Optional bool     `yaml:"optional"`
	Provides []string `yaml:"provides"`
}

// Observability 追踪与产物目录模板。
type Observability struct {
	Trace      bool   `yaml:"trace"`
	RunStore   string `yaml:"run_store"`
	CostLedger string `yaml:"cost_ledger"`
}

// StageIDs 按 YAML 顺序返回阶段 id（MVP 要求依赖阶段排在前面）。
func (d *Definition) StageIDs() []string {
	ids := make([]string, len(d.Stages))
	for i, s := range d.Stages {
		ids[i] = s.ID
	}
	return ids
}

// StageByID 按 id 查找阶段定义。
func (d *Definition) StageByID(id string) *StageDefinition {
	for i := range d.Stages {
		if d.Stages[i].ID == id {
			return &d.Stages[i]
		}
	}
	return nil
}
