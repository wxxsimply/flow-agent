package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// physicsPromptSuffix 将镜级物理字段并入 i2v/文生图 negative 与正向约束。
func physicsPromptSuffix(shot artifacts.Shot) (positive, negative string) {
	if c := strings.TrimSpace(shot.PhysicsCues); c != "" {
		positive = "，物理约束：" + c
	}
	if f := strings.TrimSpace(shot.ForbiddenPhysics); f != "" {
		negative = "，禁止：" + f
	}
	return positive, negative
}
