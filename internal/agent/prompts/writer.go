package prompts

import "fmt"

// WriterSystem 写作阶段 system prompt。
const WriterSystem = `你是连载爽文短视频的解说文案作者。根据场景目标写一段旁白正文。

要求：
1. 只输出正文，不要标题、不要 markdown、不要「场景N」等标记。
2. 口语化、节奏快，适合竖屏短视频配音。
3. 严格控制字数，不要超过 max_chars。
4. 不要复述系列设定，专注本场景剧情。`

// WriterUser 构造单场景写作用户消息。
func WriterUser(episodeNo int, sceneID int, title, goal string, maxChars int, briefExcerpt string) string {
	if maxChars <= 0 {
		maxChars = 300
	}
	return fmt.Sprintf(`第 %d 集 · 场景 %d · %s
场景目标: %s
字数上限: %d 字

本集 brief 摘要:
%s

请写本场景旁白正文。`, episodeNo, sceneID, title, goal, maxChars, briefExcerpt)
}

// WriterUserWithFix 带一致性修复说明的写作用户消息。
func WriterUserWithFix(episodeNo int, sceneID int, title, goal string, maxChars int, briefExcerpt, fixHint, characterState string) string {
	base := WriterUser(episodeNo, sceneID, title, goal, maxChars, briefExcerpt)
	if characterState != "" && characterState != "{}" {
		base += "\n\n当前角色状态 (character-state):\n" + characterState
	}
	if fixHint == "" {
		return base
	}
	return base + "\n\n【一致性修复 — 必须落实】\n" + fixHint
}
