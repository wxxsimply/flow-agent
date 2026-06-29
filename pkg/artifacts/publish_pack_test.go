package artifacts

import "testing"

func TestPublishPackValidate(t *testing.T) {
	p := &PublishPackDoc{
		EpisodeNo:   1,
		Title:       "【第1集】测试标题",
		Description: "这是一段足够长的描述文案，用于引导用户关注追更。",
		Hashtags:    []string{"小说推文", "短剧", "爽文"},
		VideoPath:   "artifacts/master.mp4",
	}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
}
