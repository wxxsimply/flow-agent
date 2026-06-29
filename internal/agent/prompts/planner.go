package prompts

import (
	"fmt"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// PlannerSystem 规划阶段 system prompt。
const PlannerSystem = `你是连载爽文短视频的策划编辑。根据系列设定与本集编号，输出本集策划。

要求：
1. 只输出一个 JSON 对象，不要 markdown 代码块，不要其它说明文字。整个回复必须是有效的单个 JSON。
2. JSON 必须包含 brief_md（字符串，Markdown 格式的 episode brief）和 hook_plan（对象）。
3. hook_plan 字段：episode_no（整数）, hook_type（字符串）, hook_line（字符串）, scene_count（整数）, scenes（数组）。
4. scenes 每项字段类型严格如下：id 必须为整数（如 1,2,3，不可写成 "1" 或 "scene-1"），title 字符串，goal 字符串，max_chars 整数（必填；所有场景 max_chars 之和须在目标字数范围内，见用户消息）。
5. scene_count 必须等于 scenes 数组长度，建议 4-6 场；单场景 max_chars 建议 120-350。
6. brief_md 须含：本集目标、情绪节奏、目标字数范围、与上集衔接。所有正文内容写在 brief_md 字符串内部（用 \n 换行），不要把任何文本放到 JSON 对象之外。
7. 竖屏解说体爽文，快节奏，结尾留 cliffhanger。`

// PlannerUser 构造 planner 用户消息。
func PlannerUser(seriesID string, episodeNo, targetSec int, bible, prevSummary, publishMetrics, nextHints string) string {
	prev := prevSummary
	if prev == "" {
		prev = "（首集或无归档，无上集摘要）"
	}
	metrics := publishMetrics
	if metrics == "" {
		metrics = "（无上集发布数据；可用 flowagent metrics set 录入）"
	}
	hints := nextHints
	if hints == "" {
		hints = "（无下集 hints；上集 learn 后会生成）"
	}
	minChars, maxChars := artifacts.ChapterCharBounds(targetSec, nil)
	return fmt.Sprintf(`系列 ID: %s
本集: 第 %d 集
目标旁白时长: 约 %d 秒（正文 %d-%d 字）

## 系列设定 (series-bible)
%s

## 上集摘要
%s

## 上集发布指标（publish-metrics）
%s

## 下集策划提示（learn 产出）
%s

请输出 JSON。brief_md 须明确回应上集 cliffhanger 与发布数据中的观众反馈（若有）。
注意：所有 scenes 的 max_chars 之和必须在 %d-%d 之间。`, seriesID, episodeNo, targetSec, minChars, maxChars, bible, prev, metrics, hints, minChars, maxChars)
}
