package agent

import (
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

func shotIdentityPromptSuffix(shot artifacts.Shot) string {
	count := shot.CharacterCount
	if count <= 0 {
		count = 1
	}
	var b strings.Builder
	if count == 1 {
		b.WriteString("，仅一名主角入画，禁止重复分身")
	}
	return b.String()
}

func shotPropLockSuffix(shot artifacts.Shot) string {
	if block := artifacts.PropLockBlock(shot.HeldProps.String()); block != "" {
		return "\n" + block
	}
	return ""
}

func shotIdentityNegSuffix(shot artifacts.Shot) string {
	count := shot.CharacterCount
	if count <= 0 {
		count = 1
	}
	return artifacts.PropsConsistencyNeg(shot.HeldProps.String(), count)
}

func keyframeBeatForShot(shot artifacts.Shot, directorUnexpanded bool) string {
	if directorUnexpanded {
		if v := strings.TrimSpace(shot.VisualPrompt); v != "" {
			return v
		}
	}
	held := strings.TrimSpace(shot.HeldProps.String())
	visual := strings.TrimSpace(shot.VisualPrompt)
	if held == "" {
		if len(shot.ActionBeats) > 0 {
			return shot.ActionBeats[0]
		}
		return visual
	}
	if len(shot.ActionBeats) == 0 {
		return visual
	}
	hands := artifacts.ParsePropHands(held + " " + visual)
	bestIdx := -1
	bestScore := 0
	for i, beat := range shot.ActionBeats {
		score := propBeatMatchScore(beat, visual, hands)
		if score > bestScore {
			bestScore = score
			bestIdx = i
		}
	}
	if bestIdx >= 0 && bestScore > 0 {
		return shot.ActionBeats[bestIdx]
	}
	return shot.ActionBeats[len(shot.ActionBeats)-1]
}

func propBeatMatchScore(beat, visual string, hands artifacts.PropHands) int {
	score := 0
	combined := beat + " " + visual
	if hands.Right != "" {
		if strings.Contains(beat, "右手") || strings.Contains(beat, hands.Right) {
			score += 2
		}
		if strings.Contains(combined, hands.Right) {
			score++
		}
	}
	if hands.Left != "" {
		if strings.Contains(beat, "左手") || strings.Contains(beat, hands.Left) {
			score += 2
		}
		if strings.Contains(combined, hands.Left) {
			score++
		}
	}
	if strings.Contains(beat, "握") || strings.Contains(beat, "持") {
		score++
	}
	return score
}
