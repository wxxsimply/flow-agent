package artifacts

import (
	"encoding/json"
	"fmt"
	"os"
)

// PublishPackDoc publish_pack_v1（与 internal/adapter/douyin.PublishPack 对齐）。
type PublishPackDoc struct {
	EpisodeNo   int      `json:"episode_no"`
	SeriesID    string   `json:"series_id,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Hashtags    []string `json:"hashtags"`
	VideoPath   string   `json:"video_path"`
	CoverPath   string   `json:"cover_path,omitempty"`
}

// Validate 校验发布包必填字段。
func (p *PublishPackDoc) Validate() error {
	if p.EpisodeNo <= 0 {
		return fmt.Errorf("episode_no required")
	}
	if len([]rune(p.Title)) == 0 || len([]rune(p.Title)) > 55 {
		return fmt.Errorf("title length invalid")
	}
	if len([]rune(p.Description)) < 10 {
		return fmt.Errorf("description too short")
	}
	if len(p.Hashtags) < 3 || len(p.Hashtags) > 15 {
		return fmt.Errorf("hashtags count %d, want 3-15", len(p.Hashtags))
	}
	if p.VideoPath == "" {
		p.VideoPath = MediaDir + "/master.mp4"
	}
	return nil
}

// LoadPublishPack 读取 publish-pack.json。
func LoadPublishPack(path string) (*PublishPackDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p PublishPackDoc
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
