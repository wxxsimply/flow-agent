// Package runctx 提供单次工作流运行的上下文与产物读写（与 runner/stage 解耦，避免循环依赖）。
package runctx

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/cost"
	"github.com/flow-agent/flow-agent/internal/provider"
	"github.com/flow-agent/flow-agent/internal/provider/llm"
	"github.com/flow-agent/flow-agent/internal/workflow"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// Context 对应计划书中的 ContentRunContext。
type Context struct {
	RunID     string // 本次运行 UUID
	TraceID   string // 追踪 ID
	Workflow  string // 工作流名
	SeriesID  string // 系列 id
	EpisodeNo int    // 集数
	Stack     string // 技术栈 profile
	PlotInput string // 微电影：用户模糊剧情（--plot）
	Creative  *artifacts.CreativeOptions
	DryRun    bool   // 是否干跑
	AutoGate  bool   // 是否自动通过人工门禁
	StopAfterStage string // 非空时在该阶段后暂停审阅

	App       *config.App
	Providers *provider.Bundle
	Def       *workflow.Definition
	Manifest     *artifacts.Manifest
	CostRecorder *cost.Recorder

	RunDir       string // runs/<run_id>/
	ArtifactsDir string // runs/<run_id>/artifacts/
	SeriesDir    string // series/<series_id>/
	Stage        string // 当前阶段 id
}

// ArtifactPath 将相对路径拼到 RunDir 下（兼容新旧产物布局）。
func (rc *Context) ArtifactPath(rel string) string {
	return artifacts.ResolvePath(rc.RunDir, rel)
}

// TargetDurationSec 目标视频时长（秒），优先级：creative > 工作流 context > stack > 默认 150。
// 低成本栈（cost_budget_cny≤5）以 stack.target_duration_sec 为硬顶，避免 UI 传入 120s 导致超支。
func (rc *Context) TargetDurationSec() int {
	target := 0
	if rc.Creative != nil && rc.Creative.TargetDurationSec > 0 {
		target = rc.Creative.TargetDurationSec
	}
	if target <= 0 && rc.Def != nil && rc.Def.Context != nil {
		if v, ok := rc.Def.Context["target_duration_sec"].(int); ok && v > 0 {
			target = v
		}
		if f, ok := rc.Def.Context["target_duration_sec"].(float64); ok && f > 0 {
			target = int(f)
		}
	}
	if target <= 0 && rc.App != nil && rc.App.Stack != nil && rc.App.Stack.TargetDurationSec > 0 {
		target = rc.App.Stack.TargetDurationSec
	}
	if target <= 0 {
		target = 150
	}
	if rc.App != nil && rc.App.Stack != nil {
		stack := rc.App.Stack
		if stack.TargetDurationSec > 0 && target > stack.TargetDurationSec {
			if stack.CostBudgetCNY > 0 && stack.CostBudgetCNY <= 5 {
				target = stack.TargetDurationSec
			}
		}
	}
	return target
}

// ManifestPath manifest.json 绝对路径。
func (rc *Context) ManifestPath() string {
	return filepath.Join(rc.RunDir, "manifest.json")
}

// CostLedgerPath cost-ledger.json 绝对路径。
func (rc *Context) CostLedgerPath() string {
	return filepath.Join(rc.RunDir, "cost-ledger.json")
}

// SaveManifest 持久化运行状态。
func (rc *Context) SaveManifest() error {
	return rc.Manifest.Save(rc.ManifestPath())
}

// WriteArtifact 写入产物文件（自动创建父目录，使用新布局路径）。
func (rc *Context) WriteArtifact(rel string, data []byte) error {
	rel = artifacts.CanonicalWriteRel(rel)
	path := filepath.Join(rc.RunDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ArtifactExists 检查产物是否已存在（兼容新旧布局）。
func (rc *Context) ArtifactExists(rel string) bool {
	return artifacts.FileExists(rc.RunDir, rel)
}

// RecordArtifact 在 manifest 中登记产物条目。
func (rc *Context) RecordArtifact(name, rel string, required bool) {
	for i, a := range rc.Manifest.Artifacts {
		if a.Path == rel {
			rc.Manifest.Artifacts[i].Name = name
			rc.Manifest.Artifacts[i].Required = required
			return
		}
	}
	rc.Manifest.Artifacts = append(rc.Manifest.Artifacts, artifacts.ArtifactEntry{
		Name:     name,
		Path:     rel,
		Required: required,
	})
}

// SetGate 记录门禁是否通过。
func (rc *Context) SetGate(id string, passed bool) {
	if rc.Manifest.Gates == nil {
		rc.Manifest.Gates = map[string]bool{}
	}
	rc.Manifest.Gates[id] = passed
}

// GatePassed 查询门禁状态。
func (rc *Context) GatePassed(id string) bool {
	return rc.Manifest.Gates[id]
}

// InitCostRecorder 按 stack 单价初始化记账器（run/resume 在 App 就绪后调用）。
func (rc *Context) InitCostRecorder() {
	if rc.App == nil {
		return
	}
	rc.CostRecorder = cost.NewRecorder(cost.RatesFromStack(rc.App.Stack))
	if rc.Manifest != nil && rc.Manifest.Cost != nil {
		rc.CostRecorder.RestoreFrom(rc.Manifest.Cost)
	}
	if data, err := os.ReadFile(rc.CostLedgerPath()); err == nil {
		var ledger artifacts.CostLedger
		if json.Unmarshal(data, &ledger) == nil {
			rc.CostRecorder.RestoreFrom(&ledger)
			if rc.Manifest.Cost != nil {
				rc.CostRecorder.SyncTo(rc.Manifest.Cost)
			}
		}
	}
}

// RecordLLM 累加 LLM 用量并同步到 manifest.cost。
func (rc *Context) RecordLLM(u llm.TokenUsage) {
	if rc.CostRecorder == nil {
		return
	}
	rc.CostRecorder.AddLLM(u)
	rc.SyncCost()
}

// RecordTTS 累加 TTS 字符数。
func (rc *Context) RecordTTS(chars int) {
	if rc.CostRecorder == nil {
		return
	}
	rc.CostRecorder.AddTTS(chars)
	rc.SyncCost()
}

// RecordImage 累加出图张数。
func (rc *Context) RecordImage(n int) {
	if rc.CostRecorder == nil {
		return
	}
	rc.CostRecorder.AddImage(n)
	rc.SyncCost()
}

// RecordVideo 累加 AI 视频秒数。
func (rc *Context) RecordVideo(seconds float64) {
	if rc.CostRecorder == nil {
		return
	}
	rc.CostRecorder.AddVideo(seconds)
	rc.SyncCost()
}

// RecordVideoAPICall 累加视频 API 调用次数。
func (rc *Context) RecordVideoAPICall(n int) {
	if rc.CostRecorder == nil {
		return
	}
	rc.CostRecorder.AddVideoAPICall(n)
	rc.SyncCost()
}

// SyncCost 将 Recorder 折算结果写入 manifest.cost。
func (rc *Context) SyncCost() {
	if rc.CostRecorder == nil || rc.Manifest == nil || rc.Manifest.Cost == nil {
		return
	}
	rc.CostRecorder.SyncTo(rc.Manifest.Cost)
}

// SaveCostLedger 将成本分项写入 cost-ledger.json。
func (rc *Context) SaveCostLedger() error {
	if rc.Manifest.Cost == nil {
		return nil
	}
	rc.Manifest.Cost.Recalc()
	b, err := json.MarshalIndent(rc.Manifest.Cost, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rc.CostLedgerPath(), b, 0o644)
}
