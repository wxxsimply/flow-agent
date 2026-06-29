package tts

import (
	"context"
	"errors"
	"testing"
)

func TestResourceGrantFallbackOnlyOnResourceError(t *testing.T) {
	primary := &stubTTS{name: "volc", err: errors.New(`403: [resource_id=volc.tts.default] requested resource not granted`)}
	secondary := &stubTTS{name: "dash", out: []byte("audio")}
	fb := NewResourceGrantFallback(primary, secondary)

	ctx := context.Background()
	req := SynthesizeRequest{SSML: "hello", Voice: "BV406_streaming"}

	data, err := fb.Synthesize(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "audio" {
		t.Fatalf("got %q", data)
	}
}

func TestResourceGrantFallbackDoesNotFallbackOnOtherErrors(t *testing.T) {
	primary := &stubTTS{name: "volc", err: errors.New("network timeout")}
	secondary := &stubTTS{name: "dash", out: []byte("audio")}
	fb := NewResourceGrantFallback(primary, secondary)

	_, err := fb.Synthesize(context.Background(), SynthesizeRequest{SSML: "hello"})
	if err == nil || err.Error() != "network timeout" {
		t.Fatalf("expected network timeout, got %v", err)
	}
}

func TestResourceGrantFallbackCachesSecondaryFailure(t *testing.T) {
	primary := &stubTTS{name: "volc", err: errors.New(`403: resource not granted`)}
	arrearage := errors.New(`400: Arrearage`)
	secondary := &stubTTS{name: "dash", err: arrearage}
	fb := NewResourceGrantFallback(primary, secondary)

	ctx := context.Background()
	req := SynthesizeRequest{SSML: "hello", Voice: "BV406_streaming"}

	_, err := fb.Synthesize(ctx, req)
	if err != arrearage {
		t.Fatalf("first call: got %v", err)
	}

	secondary.err = errors.New("should not be called")
	_, err = fb.Synthesize(ctx, req)
	if err != arrearage {
		t.Fatalf("cached call: got %v", err)
	}
}

func TestIsVolcengineResourceGrantError(t *testing.T) {
	if !IsVolcengineResourceGrantError(errors.New(`code":45000030, resource not granted`)) {
		t.Fatal("expected true")
	}
	if IsVolcengineResourceGrantError(errors.New("connection reset")) {
		t.Fatal("expected false")
	}
}
