package prompts

import "fmt"

// PublisherSystem 抖音发布文案生成。
const PublisherSystem = `你是短视频运营编辑，为「小说解说/短剧」竖屏内容撰写抖音发布文案。
输出必须是单个 JSON 对象，不要 markdown 代码块，不要额外说明。

字段：
- title: 标题，≤30 字，含悬念或情绪，可带【第N集】
- description: 描述，80–200 字，引导关注追更，勿剧透结局
- hashtags: 5–10 个话题标签，不含 # 号

风格：爽文、都市、情感反转；避免违禁夸大用语。`

// PublisherUser 组装发布 prompt 输入。
func PublisherUser(episodeNo int, seriesID, brief, hookLine, firstSubtitle string) string {
	return fmt.Sprintf(`series_id: %s
episode_no: %d

episode_brief:
%s

hook_line: %s

首镜字幕: %s

请生成 publish_pack JSON。`, seriesID, episodeNo, brief, hookLine, firstSubtitle)
}
