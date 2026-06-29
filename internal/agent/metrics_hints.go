package agent

import (
	"fmt"
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// BuildNextHintsFromMetrics 仅根据发布指标生成面向 forEpisode 的策划提示（metrics set 后刷新用）。
func BuildNextHintsFromMetrics(forEpisode int, m *artifacts.PublishMetrics) string {
	if m == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# 下集策划提示（第 %d 集）\n\n", forEpisode)
	b.WriteString("## 上集发布数据\n\n")
	b.WriteString(m.FormatForPlanner())
	b.WriteString("\n\n## 承接建议\n\n")
	if m.CompletionRate > 0 && m.CompletionRate < 0.3 {
		b.WriteString("- 完播偏低：下集前 15 秒加强钩子复述与冲突前置\n")
	} else if m.CompletionRate >= 0.4 {
		b.WriteString("- 完播尚可：延续主线并加重情绪峰值\n")
	} else {
		b.WriteString("- 承接上集 cliffhanger，首场景 10 秒内给出新信息\n")
	}
	if len(m.CommentKeywords) > 0 {
		fmt.Fprintf(&b, "- 观众热词可融入标题/开场：%s\n", strings.Join(m.CommentKeywords, "、"))
	}
	if m.Views24h > 0 {
		fmt.Fprintf(&b, "- 24h 播放 %d，标题/封面可强化同类爽点\n", m.Views24h)
	}
	return b.String()
}
