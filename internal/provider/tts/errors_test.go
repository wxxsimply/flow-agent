package tts

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestUseSilentOnBillingFailureDefaultFalse(t *testing.T) {
	os.Unsetenv("FLOWAGENT_TTS_ALLOW_SILENT")
	if UseSilentOnBillingFailure() {
		t.Fatal("expected default false")
	}
}

func TestUseSilentOnBillingFailureOptIn(t *testing.T) {
	t.Setenv("FLOWAGENT_TTS_ALLOW_SILENT", "1")
	if !UseSilentOnBillingFailure() {
		t.Fatal("expected true when FLOWAGENT_TTS_ALLOW_SILENT=1")
	}
}

func TestUserHintVolcengineResourceGrant(t *testing.T) {
	err := fmt.Errorf(`code":45000030, resource not granted`)
	hint := UserHint(err)
	if hint == "" || !strings.Contains(hint, "console.volcengine.com") || !strings.Contains(hint, "test-volcengine-tts") {
		t.Fatalf("unexpected hint: %q", hint)
	}
}
