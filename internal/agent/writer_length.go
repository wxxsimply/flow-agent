package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/flow-agent/flow-agent/internal/runctx"
	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// EnforceChapterLength 按 hook-plan 与目标时长裁剪各场景，并重新合并 chapter.md。
func EnforceChapterLength(rc *runctx.Context, plan *artifacts.HookPlan) error {
	_, maxTotal := artifacts.ChapterCharBounds(rc.TargetDurationSec(), plan)

	for _, scene := range plan.Scenes {
		maxChars := scene.MaxChars
		if maxChars <= 0 {
			maxChars = (rc.TargetDurationSec() * 5) / len(plan.Scenes)
		}
		partRel := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", scene.ID))
		path := rc.ArtifactPath(partRel)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		body := extractSceneBody(string(data))
		body = truncateRunes(strings.TrimSpace(body), maxChars)
		part := fmt.Sprintf("## Scene %d\n\n%s\n", scene.ID, body)
		if err := rc.WriteArtifact(partRel, []byte(part)); err != nil {
			return err
		}
	}

	full, err := mergeChapterParts(rc, plan)
	if err != nil {
		return err
	}

	// 若仍超总上限，按场景比例再裁一轮
	for trim := 0; trim < 3 && artifacts.CountChapterBodyRunes(full) > maxTotal; trim++ {
		if err := trimChapterProportional(rc, plan, maxTotal); err != nil {
			return err
		}
		full, err = mergeChapterParts(rc, plan)
		if err != nil {
			return err
		}
	}

	return rc.WriteArtifact("artifacts/chapter.md", []byte(full))
}

func extractSceneBody(part string) string {
	lines := strings.Split(part, "\n")
	var body []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "##") {
			continue
		}
		body = append(body, line)
	}
	return strings.TrimSpace(strings.Join(body, "\n"))
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func trimChapterProportional(rc *runctx.Context, plan *artifacts.HookPlan, maxTotal int) error {
	type sceneBody struct {
		id   int
		text string
		len  int
	}
	var bodies []sceneBody
	total := 0
	for _, scene := range plan.Scenes {
		partRel := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", scene.ID))
		data, err := os.ReadFile(rc.ArtifactPath(partRel))
		if err != nil {
			return err
		}
		text := extractSceneBody(string(data))
		l := utf8.RuneCountInString(text)
		bodies = append(bodies, sceneBody{id: scene.ID, text: text, len: l})
		total += l
	}
	if total <= maxTotal {
		return nil
	}
	for i := range bodies {
		share := float64(bodies[i].len) / float64(total)
		allow := int(float64(maxTotal) * share)
		if allow < 80 {
			allow = 80
		}
		bodies[i].text = truncateRunes(bodies[i].text, allow)
		part := fmt.Sprintf("## Scene %d\n\n%s\n", bodies[i].id, bodies[i].text)
		partRel := filepath.Join("artifacts/chapter.parts", fmt.Sprintf("scene-%02d.md", bodies[i].id))
		if err := rc.WriteArtifact(partRel, []byte(part)); err != nil {
			return err
		}
	}
	return nil
}
