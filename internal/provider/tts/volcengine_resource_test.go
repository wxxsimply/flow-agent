package tts

import "testing"

func TestResolveVolcResourceID(t *testing.T) {
	cases := []struct {
		voice, product, stackResource, want string
	}{
		{"zh_male_m191_uranus_bigtts", "doubao-speech-2.0-emotion", "", "seed-tts-2.0"},
		{"BV406_streaming", "doubao-speech-2.0-emotion", "", "seed-tts-1.0"},
		{"S_abc123", "", "", "seed-icl-2.0"},
		{"", "doubao-speech-1.0", "", "seed-tts-1.0"},
		{"custom", "", "seed-tts-2.0", "seed-tts-2.0"},
	}
	for _, tc := range cases {
		if got := ResolveVolcResourceID(tc.voice, tc.product, tc.stackResource); got != tc.want {
			t.Fatalf("ResolveVolcResourceID(%q,%q,%q)=%q want %q", tc.voice, tc.product, tc.stackResource, got, tc.want)
		}
	}
}
