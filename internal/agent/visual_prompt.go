package agent

import (
	"strings"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/agent/skills"
	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// shotImagePrompt 文生图：2D/3D 风格 + 画面约束。
func shotImagePrompt(rc *runctx.Context, shot artifacts.Shot) string {
	if (IsDirectorRun(rc) || shot.UserSource) && !shot.Expanded {
		return shotDirectImagePrompt(rc, shot)
	}
	preset := VisualPresetForRun(rc)
	p := strings.TrimSpace(shot.VisualPrompt)
	if p == "" {
		spec := config.MediaSpecFromCreative(rc.Creative)
		p = spec.LabelZH + "电影画面"
	}
	s := strings.TrimSpace(preset.ImageStylePrefix) + p
	if bg := strings.TrimSpace(shot.SceneBackground); bg != "" {
		s += "，【场景】" + bg
	}
	if pos, _ := physicsPromptSuffix(shot); pos != "" {
		s += pos
	}
	s += shotIdentityPromptSuffix(shot)
	if block := propAppearanceBlock(rc, shot); block != "" {
		s += "\n" + block
	}
	if block := propViewLockBlock(rc, shot); block != "" {
		s += "\n" + block
	}
	s += shotPropLockSuffix(shot)
	if block := characterAppearanceBlock(rc); block != "" {
		s += "\n" + block
	}
	if block := characterViewLockBlock(rc, shot.VisualPrompt); block != "" {
		s += "\n" + block
	}
	neg := preset.ImageNegative
	if _, fp := physicsPromptSuffix(shot); fp != "" {
		neg += strings.TrimPrefix(fp, "，禁止：")
	}
	if idNeg := shotIdentityNegSuffix(shot); idNeg != "" {
		neg += "，" + idNeg
	}
	return s + " [NEG] " + neg
}

// shotDirectImagePrompt director 模式：用户原文 + 景别，不混入 LLM 扩写。
func shotDirectImagePrompt(rc *runctx.Context, shot artifacts.Shot) string {
	preset := VisualPresetForRun(rc)
	p := strings.TrimSpace(shot.VisualPrompt)
	if p == "" {
		p = strings.TrimSpace(shot.Narration)
	}
	hint := artifacts.ShotSizePromptHint(shot.ShotSize)
	s := strings.TrimSpace(preset.ImageStylePrefix) + "\n" + hint + "，" + p
	s += "，严格对应用户描述的单帧画面，禁止添加未描述元素"
	if block := characterAppearanceBlock(rc); block != "" {
		s += "\n" + block
	}
	if block := propAppearanceBlock(rc, shot); block != "" {
		s += "\n" + block
	}
	if block := characterViewLockBlock(rc, shot.VisualPrompt); block != "" {
		s += "\n" + block
	}
	if block := propViewLockBlock(rc, shot); block != "" {
		s += "\n" + block
	}
	s += shotIdentityPromptSuffix(shot)
	s += shotPropLockSuffix(shot)
	neg := preset.ImageNegative
	if idNeg := shotIdentityNegSuffix(shot); idNeg != "" {
		neg += "，" + idNeg
	}
	return s + " [NEG] " + neg
}

// shotMotionPrompt 图生视频运镜：强调物理合理、避免穿模。
func shotMotionPrompt(rc *runctx.Context, vidCfg config.StackVideoConfig, shot artifacts.Shot, prevShot *artifacts.Shot) string {
	preset := VisualPresetForRun(rc)
	motionSuffix := vidCfg.MotionPromptSuffix
	p := strings.TrimSpace(shot.VisualPrompt)
	if p == "" {
		p = strings.TrimSpace(shot.Narration)
	}
	if (IsDirectorRun(rc) || shot.UserSource) && !shot.Expanded {
		p = artifacts.ShotSizePromptHint(shot.ShotSize) + "，" + p
		p += "，镜头内仅呈现上述画面，小幅运镜，禁止新增场景或角色"
	} else if shot.Expanded && shot.ShotSize != "" {
		p = artifacts.ShotSizePromptHint(shot.ShotSize) + "，" + p
	}
	p += preset.VisualGuard + config.GlobalVisualGuard
	p += shotIdentityPromptSuffix(shot)
	if block := propAppearanceBlock(rc, shot); block != "" {
		p += "\n" + block
	}
	if block := propViewLockBlock(rc, shot); block != "" {
		p += "\n" + block
	}
	p += shotPropLockSuffix(shot)
	if prevShot != nil {
		if shouldLinkToPreviousShot(*prevShot, shot) {
			p += crossShotContinuitySuffix(*prevShot, shot)
		} else {
			p += "，本镜为场景切换，独立构图与运镜，仅保持主角外形/服装一致"
		}
	}
	if block := characterViewLockBlock(rc, p); block != "" {
		p += "\n" + block
	}
	if block := propViewLockBlock(rc, shot); block != "" {
		p += "\n" + block
	}
	p += "，镜头内角色数量不变"
	if pos := artifacts.PropsHandConsistencyPos(shot.HeldProps.String()); pos != "" {
		p += pos
		p += "，小幅运镜，持物手位移最小"
	}
	if pos, _ := physicsPromptSuffix(shot); pos != "" {
		p += pos
	}
	p += preset.VideoMotionSuffix
	if block := skills.MotionPromptBlockForProvider(vidCfg.Provider); block != "" {
		p += block
	}
	if s := strings.TrimSpace(motionSuffix); s != "" {
		p += s
	}
	if _, neg := physicsPromptSuffix(shot); neg != "" {
		p += neg
	}
	if idNeg := shotIdentityNegSuffix(shot); idNeg != "" {
		p += "，禁止：" + idNeg
	}
	maxRunes := defaultMotionPromptMaxRunes
	if isSoraVideoProvider(vidCfg) {
		maxRunes = soraMotionPromptMaxRunes
	}
	return truncateMotionPrompt(p, maxRunes)
}

const defaultMotionPromptMaxRunes = 800

func truncateMotionPrompt(p string, maxRunes int) string {
	if maxRunes <= 0 {
		maxRunes = defaultMotionPromptMaxRunes
	}
	if utf8.RuneCountInString(p) <= maxRunes {
		return p
	}
	runes := []rune(p)
	return string(runes[:maxRunes]) + "…"
}
