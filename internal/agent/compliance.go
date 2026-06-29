package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flow-agent/flow-agent/internal/adapter/douyin"
	"github.com/flow-agent/flow-agent/internal/compliance"
	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// RunCompliance 扫描 chapter / storyboard / 发布文案，输出 compliance-report.json。
func RunCompliance(rc *runctx.Context) error {
	configDir := "config"
	if rc.App != nil && rc.App.ConfigDir != "" {
		configDir = rc.App.ConfigDir
	}

	entries, err := compliance.LoadWordLists(configDir)
	if err != nil {
		return fmt.Errorf("load compliance word lists: %w", err)
	}

	sources, err := collectComplianceSources(rc)
	if err != nil {
		return err
	}

	report := compliance.ScanSources(entries, sources)
	report.EpisodeNo = rc.EpisodeNo
	report.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	if err := report.Validate(rc.EpisodeNo); err != nil {
		return err
	}

	path := rc.ArtifactPath("artifacts/compliance-report.json")
	if err := report.Save(path); err != nil {
		return err
	}
	rc.RecordArtifact("compliance-report.json", "artifacts/compliance-report.json", true)
	return nil
}

func collectComplianceSources(rc *runctx.Context) ([]compliance.TextSource, error) {
	var sources []compliance.TextSource

	addFile := func(name, rel string) error {
		if !rc.ArtifactExists(rel) {
			return nil
		}
		data, err := os.ReadFile(rc.ArtifactPath(rel))
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		sources = append(sources, compliance.TextSource{Name: name, Text: string(data)})
		return nil
	}

	if err := addFile("chapter.md", "artifacts/chapter.md"); err != nil {
		return nil, err
	}
	if err := addFile("episode-brief.md", "artifacts/episode-brief.md"); err != nil {
		return nil, err
	}
	if err := addFile("narration.ssml", "artifacts/narration.ssml"); err != nil {
		return nil, err
	}

	if rc.ArtifactExists("artifacts/storyboard.json") {
		sb, err := artifacts.LoadStoryboard(rc.ArtifactPath("artifacts/storyboard.json"))
		if err != nil {
			return nil, fmt.Errorf("load storyboard: %w", err)
		}
		for i, shot := range sb.Shots {
			prefix := fmt.Sprintf("storyboard.shots[%d]", i)
			if t := strings.TrimSpace(shot.Subtitle); t != "" {
				sources = append(sources, compliance.TextSource{
					Name: prefix + ".subtitle",
					Text: t,
				})
			}
			if t := strings.TrimSpace(shot.Narration); t != "" {
				sources = append(sources, compliance.TextSource{
					Name: prefix + ".narration",
					Text: t,
				})
			}
		}
	}

	if rc.ArtifactExists("artifacts/publish-pack.json") {
		data, err := os.ReadFile(rc.ArtifactPath("artifacts/publish-pack.json"))
		if err != nil {
			return nil, fmt.Errorf("read publish-pack: %w", err)
		}
		var pack douyin.PublishPack
		if err := json.Unmarshal(data, &pack); err != nil {
			return nil, fmt.Errorf("parse publish-pack: %w", err)
		}
		if t := strings.TrimSpace(pack.Title); t != "" {
			sources = append(sources, compliance.TextSource{Name: "publish-pack.title", Text: t})
		}
		if t := strings.TrimSpace(pack.Description); t != "" {
			sources = append(sources, compliance.TextSource{Name: "publish-pack.description", Text: t})
		}
		for i, tag := range pack.Hashtags {
			if t := strings.TrimSpace(tag); t != "" {
				sources = append(sources, compliance.TextSource{
					Name: fmt.Sprintf("publish-pack.hashtags[%d]", i),
					Text: t,
				})
			}
		}
	}

	return sources, nil
}
