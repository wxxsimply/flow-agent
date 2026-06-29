package agent

import (
	"fmt"
	"strings"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// ShotKeyframeImagePrompt 单帧关键帧文生图：beat + 镜级物理 + 角色锁定。
func ShotKeyframeImagePrompt(rc *runctx.Context, shot artifacts.Shot, beat string, prevShot *artifacts.Shot, hardChain bool) string {
	if beat == "" {
		return shotImagePrompt(rc, shot)
	}
	preset := VisualPresetForRun(rc)
	p := strings.TrimSpace(beat)
	if p == "" {
		p = strings.TrimSpace(shot.VisualPrompt)
	}
	if p == "" {
		p = "电影画面关键帧"
	}
	s := strings.TrimSpace(preset.ImageStylePrefix) + "\n" + p
	s += "，单帧关键姿态，禁止多动作叠加"
	if shot.Expanded || IsDirectorRun(rc) {
		if hint := artifacts.ShotSizePromptHint(shot.ShotSize); hint != "" {
			s += "，" + hint
		}
	}
	if pos, _ := physicsPromptSuffix(shot); pos != "" {
		s += pos
	}
	if block := characterAppearanceBlock(rc); block != "" {
		s += "\n" + block
	}
	if block := propAppearanceBlock(rc, shot); block != "" {
		s += "\n" + block
	}
	if block := characterViewLockBlock(rc, p); block != "" {
		s += "\n" + block
	}
	if block := propViewLockBlock(rc, shot); block != "" {
		s += "\n" + block
	}
	s += shotIdentityPromptSuffix(shot)
	s += shotPropLockSuffix(shot)
	if prevShot != nil && shouldLinkToPreviousShot(*prevShot, shot) {
		if hardChain {
			s += crossShotContinuitySuffixHard(*prevShot, shot)
		} else {
			s += crossShotContinuitySuffix(*prevShot, shot)
		}
	}
	neg := preset.ImageNegative
	if _, fp := physicsPromptSuffix(shot); fp != "" {
		neg += "，" + strings.TrimPrefix(fp, "，禁止：")
	}
	if idNeg := shotIdentityNegSuffix(shot); idNeg != "" {
		neg += "，" + idNeg
	}
	s += " [NEG] " + neg
	return s
}

// ShotSegmentMotionPrompt 关键帧之间的图生视频 prompt。
func ShotSegmentMotionPrompt(rc *runctx.Context, vidCfg config.StackVideoConfig, shot artifacts.Shot, fromBeat, toBeat string) string {
	base := shotMotionPrompt(rc, vidCfg, shot, nil)
	transition := fmt.Sprintf("从「%s」平滑过渡到「%s」", strings.TrimSpace(fromBeat), strings.TrimSpace(toBeat))
	return base + "，" + transition + "，动作连贯幅度小，禁止瞬间位移与头部身体朝向矛盾"
}
