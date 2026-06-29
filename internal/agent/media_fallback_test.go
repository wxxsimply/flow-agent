package agent

import (
	"errors"
	"fmt"
	"testing"

	"github.com/flow-agent/flow-agent/internal/config"
	"github.com/flow-agent/flow-agent/internal/runctx"
)

func TestIsArrearage(t *testing.T) {
	err := fmt.Errorf(`wan create: http 400: {"code":"Arrearage"}`)
	if !isArrearage(err) {
		t.Fatal("expected arrearage")
	}
	err2 := fmt.Errorf(`volcengine ark: 403: AccountOverdueError overdue balance`)
	if !isArrearage(err2) {
		t.Fatal("expected volcengine overdue")
	}
	if isArrearage(errors.New("timeout")) {
		t.Fatal("timeout is not arrearage")
	}
}

func TestImageToVideoShouldKenBurns_arrearageRequireVideo(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Video: map[string]any{"require_video": true},
			},
		},
	}
	err := errors.New(`{"code":"Arrearage"}`)
	if imageToVideoShouldKenBurns(rc, err) {
		t.Fatal("arrearage on require_video stack should not ken burns")
	}
}

func TestImageToVideoShouldKenBurns_arrearageOptionalVideo(t *testing.T) {
	rc := &runctx.Context{
		App: &config.App{
			Stack: &config.Stack{
				Video: map[string]any{"require_video": false},
			},
		},
	}
	err := errors.New(`{"code":"Arrearage"}`)
	if !imageToVideoShouldKenBurns(rc, err) {
		t.Fatal("arrearage without require_video may ken burns")
	}
}
