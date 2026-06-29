package tts

import (
	"context"
	"errors"
	"testing"
)

type stubTTS struct {
	name string
	err  error
	out  []byte
}

func (s *stubTTS) Synthesize(ctx context.Context, req SynthesizeRequest) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func TestFallbackUsesSecondaryAfterPrimaryFails(t *testing.T) {
	primary := &stubTTS{name: "volc", err: errors.New("403 forbidden")}
	secondary := &stubTTS{name: "dash", out: []byte("audio")}
	fb := NewFallback(primary, secondary)

	ctx := context.Background()
	req := SynthesizeRequest{SSML: "hello", Voice: "BV406_streaming"}

	data, err := fb.Synthesize(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "audio" {
		t.Fatalf("got %q", data)
	}

	// second call should not hit primary again
	primary.err = errors.New("should not be called")
	data, err = fb.Synthesize(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "audio" {
		t.Fatalf("got %q", data)
	}
}
