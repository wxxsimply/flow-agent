package config



import "testing"



func TestVoicePresetByID(t *testing.T) {

	p := VoicePresetByID("epic_male")

	if p.VoiceID != "zh_male_m191_uranus_bigtts" || p.SpeedRatio >= 1.0 {

		t.Fatalf("epic_male: voice=%s speed=%v", p.VoiceID, p.SpeedRatio)

	}

	v, speed := ResolveNarratorVoice("documentary_male", "default")

	if v != "zh_male_liufei_uranus_bigtts" || speed <= 0 {

		t.Fatalf("documentary: voice=%s speed=%v", v, speed)

	}

}



func TestDashScopeVoiceFor(t *testing.T) {

	cases := map[string]string{

		"BV406_streaming":                "longanyang",

		"BV001_streaming":                "longwan",

		"zh_male_m191_uranus_bigtts":     "longanyang",

		"zh_female_vv_uranus_bigtts":     "longwan",

		"longanyang":                     "longanyang",

		"":                               "longanyang",

	}

	for in, want := range cases {

		if got := DashScopeVoiceFor(in); got != want {

			t.Fatalf("DashScopeVoiceFor(%q)=%q want %q", in, got, want)

		}

	}

}

