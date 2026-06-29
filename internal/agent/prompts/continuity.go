package prompts

import "fmt"

// ContinuitySystem 一致性校验 system prompt。
const ContinuitySystem = `你是连载小说的设定编辑。对比「系列设定 bible」「角色状态」「伏笔检索」与「本集正文」，找出人设、时间线、伏笔矛盾。

要求：
1. 只输出一个 JSON 对象，不要 markdown 代码块。
2. 字段：episode_no, critical_count, warning_count, issues, passed, character_state_patch（可选对象）。
3. issues 数组项：severity（critical|warning）, category, scene_id（整数，对应场景编号）, message, suggestion。
4. critical 仅限「事实性矛盾」：与 bible 设定直接冲突、时间线不可能、人名/身份错误、已埋伏笔被违反。
5. 以下只能标为 warning，不得标 critical：文笔、节奏、心理描写多少、情绪过渡是否够细腻、人设「表现力」建议。
6. 无事实性问题时 issues 为空或仅 warning，passed 为 true。
7. character_state_patch 严格规则（违反则不要输出此字段）：
   - 只能写「本集正文实际发生且已明示」的角色状态变化；
   - 禁止预写未揭示的未来剧情（例如 logline 说"三年后真相揭开"，本集就不得在 known_secrets 写入该真相）；
   - 禁止把 bible 已有的静态人设（traits、role）作为 patch 重复写入；
   - 不确定时宁可省略整个 character_state_patch 字段，也不要凭推测填写。`

// ContinuityUser 构造 continuity 用户消息。
func ContinuityUser(episodeNo int, bible, characterState, foreshadowHits, chapter string) string {
	if characterState == "" {
		characterState = "（尚无 character-state.json）"
	}
	if foreshadowHits == "" {
		foreshadowHits = "（无检索命中）"
	}
	return fmt.Sprintf(`本集: 第 %d 集

## 系列 bible
%s

## 角色状态 (character-state.json)
%s

## 伏笔 / 设定检索
%s

## 本集正文 (chapter.md)
%s

请输出 continuity JSON。`, episodeNo, bible, characterState, foreshadowHits, chapter)
}
