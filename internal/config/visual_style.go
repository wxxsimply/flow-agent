package config

import (
	"strings"

	"github.com/flow-agent/flow-agent/pkg/artifacts"
)

// VisualStylePreset 2D/3D 与画面一致性约束。
type VisualStylePreset struct {
	Name              string
	ImageStylePrefix  string
	ImageNegative     string
	VideoMotionSuffix string
	VisualGuard       string
	StoryboardHint    string
}

// AnimationStylePreset 返回 2d 或 3d 风格包。
func AnimationStylePreset(style string) VisualStylePreset {
	if strings.EqualFold(strings.TrimSpace(style), "3d") {
		return style3D()
	}
	return style2D()
}

// VisualPresetFromCreative 根据用户选项返回视觉预设（含主题/横竖屏）。
func VisualPresetFromCreative(c *artifacts.CreativeOptions) VisualStylePreset {
	style := "2d"
	if c != nil {
		style = c.AnimationStyle
	}
	p := AnimationStylePreset(style)
	theme := "generic"
	if c != nil && strings.TrimSpace(c.VisualTheme) != "" {
		theme = strings.ToLower(strings.TrimSpace(c.VisualTheme))
	}
	switch theme {
	case "generic", "default", "normal", "通用":
		p.StoryboardHint = "画面风格：通用动画插画，干净构图，非写实照片感。"
	case "cinematic", "电影":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "电影写实2D分镜")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "电影写实3D渲染")
		p.StoryboardHint = "画面风格：电影写实，自然光影与景深，避免卡通夸张。"
	case "anime", "日系", "番剧":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "日系赛璐璐2D动画")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "日系3D动画")
		p.StoryboardHint = "画面风格：日系动漫，赛璐璐上色，清晰线稿，番剧镜头语言。"
	case "cartoon", "卡通":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "欧美卡通2D动画")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "欧美卡通3D动画")
		p.StoryboardHint = "画面风格：欧美卡通，造型夸张，色彩饱和，表情生动。"
	case "ink_wash", "水墨", "ink":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "水墨国风2D插画")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "水墨国风3D动画")
		p.StoryboardHint = "画面风格：水墨国风，留白、晕染、笔意，避免写实照片感。"
	case "cyberpunk", "赛博朋克", "cyber":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "赛博朋克2D动画")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "赛博朋克3D动画")
		p.StoryboardHint = "画面风格：赛博朋克，霓虹、雨夜、高对比，避免田园写实。"
	case "ghibli", "吉卜力", "治愈":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "治愈手绘2D动画")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "治愈手绘3D动画")
		p.StoryboardHint = "画面风格：治愈手绘，柔和光影与自然细节，日常温情。"
	case "wuxia", "武侠", "古风":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "国风武侠2D动画")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "国风武侠3D动画")
		p.StoryboardHint = "画面风格：国风武侠，飘逸衣袂、竹林山雾，避免现代都市元素。"
	case "noir", "悬疑":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "黑色电影noir风格2D")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "黑色电影noir风格3D")
		p.StoryboardHint = "画面风格：悬疑 noir，低调明暗对比，阴影与张力，避免明亮卡通。"
	case "commercial", "广告":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "商业广告级2D视觉")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "商业广告级3D视觉")
		p.StoryboardHint = "画面风格：商业广告，干净构图、高质感、主体突出，适合展示。"
	case "sketch", "素描":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "手绘素描线稿2D")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "手绘素描质感3D")
		p.StoryboardHint = "画面风格：手绘素描，线稿与排线质感，艺术感强。"
	case "pixel", "像素":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "像素复古2D")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "低多边形像素风3D")
		p.StoryboardHint = "画面风格：像素复古，8-bit 块面与有限色板，怀旧游戏感。"
	case "arknights":
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "2D动画电影感", "战术插画风格2D分镜")
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "3D动画电影感", "战术插画风格3D动画")
		p.StoryboardHint = "画面风格：战术插画（干净线条、清晰分层），避免写实照片感。"
	default:
		p.StoryboardHint = "画面风格：通用动画插画，干净构图，非写实照片感。"
	}
	// 横竖屏
	if c != nil && strings.EqualFold(strings.TrimSpace(c.Orientation), "landscape") {
		p.ImageStylePrefix = strings.ReplaceAll(p.ImageStylePrefix, "竖屏9:16", "横屏16:9")
		p.ImageNegative = strings.ReplaceAll(p.ImageNegative, "横构图", "竖构图")
	}
	return p
}

func style2D() VisualStylePreset {
	return VisualStylePreset{
		Name: "2d",
		ImageStylePrefix: `[STYLE] 2D动画电影感，竖屏9:16，赛璐璐或高质量插画，线条清晰，色块分明，光影层次分明
[NEG] 写实照片感、3D渲染感、文字水印、多余手指、肢体畸形`,
		ImageNegative: "穿模, 身体穿透, 肢体融合, 多余手臂, 脸部扭曲, 低清晰度, 横构图, 水印, duplicate character, two identical persons, extra weapons, 多余武器, 克隆人",
		VideoMotionSuffix: "，2D动画运镜，动作幅度适中，表情清晰，禁止写实3D质感",
		VisualGuard: "，空间关系合理，物体不穿透，人物比例正确，动作符合物理常识，单镜单焦点",
		StoryboardHint: "画面风格：2D动画/插画电影，非写实3D。",
	}
}

func style3D() VisualStylePreset {
	return VisualStylePreset{
		Name: "3d",
		ImageStylePrefix: `[STYLE] 3D动画电影感，竖屏9:16，皮克斯/国漫3D质感，材质细腻，体积光
[NEG] 2D平面插画、赛璐璐、手绘线稿感、文字水印、多余手指`,
		ImageNegative: "穿模, 穿帮, 身体穿透, 肢体穿插, 多余肢体, 脸部崩坏, 低模糊面, 横构图, duplicate character, extra weapons",
		VideoMotionSuffix: "，3D动画运镜，角色动作连贯自然，禁止夸张扭曲导致穿模",
		VisualGuard: "，刚体不穿透，碰撞合理，重力方向一致，镜头内空间逻辑自洽，避免复杂多人肢体交叠",
		StoryboardHint: "画面风格：3D动画电影，注意刚体与空间逻辑。",
	}
}

// GlobalVisualGuard 所有风格共用的物理/逻辑约束（写入分镜与视频 prompt）。
const GlobalVisualGuard = "，禁止穿模与物体穿透，禁止违反重力与空间逻辑，禁止肢体数量错误，单镜内动作简洁可完成，禁止同一人物重复入镜，道具数量须与描述一致"
