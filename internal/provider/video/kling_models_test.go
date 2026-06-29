package video



import "testing"



func TestNormalizeKlingImage2VideoModel(t *testing.T) {

	if got := NormalizeKlingImage2VideoModel("kling-v2-5-turbo"); got != "kling-v2-1" {

		t.Fatalf("image2video turbo -> got %s", got)

	}

	if got := NormalizeKlingImage2VideoModel(""); got != "kling-v2-1" {

		t.Fatalf("empty image -> %s", got)

	}

}



func TestNormalizeKlingText2VideoModel(t *testing.T) {

	if got := NormalizeKlingText2VideoModel("kling-v2-5-turbo"); got != "kling-v1-6" {

		t.Fatalf("text2video turbo -> got %s", got)

	}

	if got := NormalizeKlingText2VideoModel("kling-v2-5"); got != "kling-v1-6" {

		t.Fatalf("text2video v2-5 alias -> got %s", got)

	}

	if got := NormalizeKlingText2VideoModel(""); got != "kling-v1-6" {

		t.Fatalf("empty text -> %s", got)

	}

}



func TestText2VideoModelsToTry(t *testing.T) {

	models := Text2VideoModelsToTry("kling-v2-5")

	if models[0] != "kling-v1-6" {

		t.Fatalf("first=%s want kling-v1-6", models[0])

	}

}


