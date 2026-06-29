package video



import "strings"



// 图生视频 model_name（可灵 OpenAPI image2video）。

var klingImage2VideoAliases = map[string]string{

	"kling-v2-5-turbo": "kling-v2-1",

	"kling-v2-5":       "kling-v2-1",

	"kling-v2-6":       "kling-v2-1",

	"kling-v2-master":  "kling-v2-1-master",

}



// 文生视频 model_name（可灵 OpenAPI text2video；v2-1 / v2-5 在多数国际账号上无效）。

var klingText2VideoAliases = map[string]string{

	"kling-v2-5-turbo":  "kling-v1-6",

	"kling-v2-5":        "kling-v1-6",

	"kling-v2-6":        "kling-v2-6",

	"kling-v2-1":        "kling-v1-6",

	"kling-v2-1-master": "kling-v2-1-master",

	"kling-v3-0":        "kling-v3-0",

}



// Text2VideoModelFallbacks 文生视频 model 回退顺序（探测与运行时共用）。

var Text2VideoModelFallbacks = []string{

	"kling-v1-6",

	"kling-v1-5",

	"kling-v2-6",

	"kling-v2-1-master",

	"kling-v2-master",

	"kling-v3-0",

}



// NormalizeKlingImage2VideoModel 映射 stack 配置名为图生视频 API model_name。

func NormalizeKlingImage2VideoModel(name string) string {

	return normalizeKlingModel(name, klingImage2VideoAliases, "kling-v2-1")

}



// NormalizeKlingText2VideoModel 映射 stack 配置名为文生视频 API model_name。

func NormalizeKlingText2VideoModel(name string) string {

	return normalizeKlingModel(name, klingText2VideoAliases, "kling-v1-6")

}



// NormalizeKlingModel 兼容旧调用，默认图生视频映射。

func NormalizeKlingModel(name string) string {

	return NormalizeKlingImage2VideoModel(name)

}



// Text2VideoModelsToTry 返回去重后的文生视频 model 尝试列表（首选 preferred）。

func Text2VideoModelsToTry(preferred string) []string {

	seen := map[string]bool{}

	var out []string

	add := func(m string) {

		m = strings.TrimSpace(m)

		if m == "" || seen[m] {

			return

		}

		seen[m] = true

		out = append(out, m)

	}

	add(NormalizeKlingText2VideoModel(preferred))

	for _, m := range Text2VideoModelFallbacks {

		add(m)

	}

	return out

}



func normalizeKlingModel(name string, aliases map[string]string, def string) string {

	n := strings.TrimSpace(name)

	if n == "" {

		return def

	}

	key := strings.ToLower(n)

	if mapped, ok := aliases[key]; ok {

		return mapped

	}

	return n

}


