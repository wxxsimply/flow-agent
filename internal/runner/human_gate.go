package runner

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/internal/workflow"
)

// deferHumanGateForReview Web/CLI 在 stop-after 暂停时，brief 确认改由审阅页或 resume 完成。
func deferHumanGateForReview(rc *runctx.Context, stage *workflow.StageDefinition, g workflow.GateDefinition) bool {
	if rc == nil || stage == nil {
		return false
	}
	if rc.StopAfterStage == "" || rc.StopAfterStage != stage.ID {
		return false
	}
	if g.ID == "brief_confirmed" {
		return true
	}
	return g.Skippable
}

// PromptHumanGates 在无 --auto-gate 时交互确认人工门禁。
func PromptHumanGates(rc *runctx.Context, stage *workflow.StageDefinition) error {
	if rc.AutoGate {
		return nil
	}
	for _, g := range stage.Gates {
		if g.Type != "human" {
			continue
		}
		if deferHumanGateForReview(rc, stage, g) {
			continue
		}
		if rc.GatePassed(g.ID) {
			continue
		}
		msg := humanGateMessage(g.ID, stage.ID)
		fmt.Fprintf(os.Stdout, "\n[human gate] %s\n%s\nApprove? [y/N]: ", g.ID, msg)
		reader := bufio.NewReader(os.Stdin)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" {
			return fmt.Errorf("human gate %q not approved", g.ID)
		}
		rc.SetGate(g.ID, true)
		if err := rc.SaveManifest(); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "approved:", g.ID)
	}
	return nil
}

func humanGateMessage(gateID, stageID string) string {
	switch gateID {
	case "outline_confirmed":
		return fmt.Sprintf("Stage %q: 请确认 episode-brief.md / hook-plan.json 无误。", stageID)
	case "brief_confirmed":
		return fmt.Sprintf("Stage %q: 请确认 shot-language-brief.md 与 storyboard.json 无误后再进入 produce。", stageID)
	case "final_cut_approved":
		return fmt.Sprintf("Stage %q: 请预览 artifacts/master.mp4 与分镜，确认成片可发布。", stageID)
	case "publish_authorized":
		return fmt.Sprintf("Stage %q: 确认 publish-pack.json 标题/描述/话题，授权发布。", stageID)
	default:
		return fmt.Sprintf("Stage %q: gate %q", stageID, gateID)
	}
}
