package volcengine

import (
	"fmt"
	"testing"
)
func TestSeedanceModelFallbacks(t *testing.T) {
	models := SeedanceModelFallbacks("doubao-seedance-2-0-260128")
	if len(models) < 2 || models[0] != "doubao-seedance-2-0-260128" {
		t.Fatalf("fallbacks=%v", models)
	}
	if models[1] != "doubao-seedance-1-5-pro-251215" {
		t.Fatalf("second=%q", models[1])
	}
}

func TestIsModelNotOpen(t *testing.T) {
	err := fmt.Errorf(`volcengine ark: 404: {"code":"ModelNotOpen","message":"has not activated the model"}`)
	if !IsModelNotOpen(err) {
		t.Fatal("expected ModelNotOpen")
	}
}

func TestIsVolcengineFatal(t *testing.T) {
	err := fmt.Errorf(`volcengine ark: 403 Forbidden: {"error":{"code":"AccountOverdueError","message":"overdue balance"}}`)
	if !IsVolcengineFatal(err) {
		t.Fatal("expected fatal overdue error")
	}
	if IsVolcengineRetryable(err) {
		t.Fatal("overdue should not be retryable")
	}
}

func TestClampSeedanceDuration(t *testing.T) {
	if got := ClampSeedanceDuration("doubao-seedance-2-0-260128", 20); got != 15 {
		t.Fatalf("2.0 max %d", got)
	}
	if got := ClampSeedanceDuration("doubao-seedance-2-0-fast-260128", 20); got != 12 {
		t.Fatalf("2.0 fast max %d", got)
	}
	if got := ClampSeedanceDuration("", 2); got != 4 {
		t.Fatalf("min %d", got)
	}
}

func TestSeedreamSize(t *testing.T) {
	if got := SeedreamSize(1080, 1920, "9:16"); got != "1080x1920" {
		t.Fatalf("got %q", got)
	}
}
