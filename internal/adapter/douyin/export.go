// Package douyin 抖音发布适配（MVP 仅导出 publish-pack，后续可接 OpenAPI）。
package douyin

// PublishPack 手机端发布或开放平台草稿所需的元数据（publish_pack_v1）。
type PublishPack struct {
	EpisodeNo   int      `json:"episode_no,omitempty"`
	SeriesID    string   `json:"series_id,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Hashtags    []string `json:"hashtags"`
	VideoPath   string   `json:"video_path"`           // 相对 run 目录
	CoverPath   string   `json:"cover_path,omitempty"` // 封面帧
}
