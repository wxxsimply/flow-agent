package agent

import (
	"errors"
	"os"
	"testing"

	"github.com/flow-agent/flow-agent/internal/runctx"
)

func TestShouldSkipI2V_envOnly(t *testing.T) {
	rc := &runctx.Context{RunID: "test-run-skip-i2v"}
	initProduceState(rc)
	defer clearProduceState(rc)

	if shouldSkipI2V(rc) {
		t.Fatal("expected i2v enabled by default")
	}
	markSkipI2V(rc, "volcengine overdue")
	if shouldSkipI2V(rc) {
		t.Fatal("markSkipI2V must not disable i2v for whole run")
	}
	t.Setenv("FLOWAGENT_SKIP_I2V", "1")
	if !shouldSkipI2V(rc) {
		t.Fatal("expected skip when FLOWAGENT_SKIP_I2V set")
	}
	os.Unsetenv("FLOWAGENT_SKIP_I2V")
}

func TestIsWanFatal(t *testing.T) {
	if !isWanFatal(errors.New(`wan create: http 403: {"code":"AccessDenied"}`)) {
		t.Fatal("expected AccessDenied to be fatal")
	}
	if isWanFatal(errors.New("timeout")) {
		t.Fatal("expected timeout not fatal")
	}
}

func TestIsArrearage_detectsBothProviders(t *testing.T) {
	if !isArrearage(errors.New(`code":"Arrearage"`)) {
		t.Fatal("dashscope arrearage")
	}
	if !isArrearage(errors.New(`AccountOverdueError`)) {
		t.Fatal("volcengine overdue")
	}
}
