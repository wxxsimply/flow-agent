package prompts

import "fmt"

// BriefCoverageDimensions 扩写必须覆盖的维度（约 3000 字内写全）。
const BriefCoverageDimensions = `扩写须面面俱到、信息可拍摄，至少覆盖：
1. 人物与剧情：核心矛盾、人物关系、情绪弧线、对白要点；
2. 叙事功能：铺垫/发展/高潮/收束各段须标明承担的剧情节拍，镜头语言须服务当前节拍（禁止与旁白情绪无关的运镜）；
3. 蒙太奇节奏：建立镜-反应镜-高潮镜比例合理，场景切换写清硬切/推近/摇镜过渡；
4. 整体形象：年龄气质、身材体态、发型、五官神态、性格外显；
5. 服装细节：上装/下装/鞋、材质配色、配饰、污渍或战损；全片使用相同锁定词；
6. 镜头语言：景别、角度、运镜（单镜一种主运动）、构图、屏幕方向，并写明「为何服务本段剧情」；
7. 表演细节：预备-动作-收势、微表情、眼神、重心、与环境互动；
8. 画面场景：时代地点、室内外、光影色调、天气、前中后景与道具；
9. 画面精致度：主光方向、色温、景深、材质纹理、抗 AI 伪影；
10. 连续性：对照 continuity-ledger，服装/道具/光线跨段一致。` + AnimationCraftPerformance + AnimationCraftVisualPolish + AnimationCraftPhysicsPrompt

// OpeningShotBriefSegmentSystem 分段扩写 system prompt（段数与总字数可配置）。
func OpeningShotBriefSegmentSystem(totalSegments, targetRunes int) string {
	if totalSegments <= 0 {
		totalSegments = 3
	}
	if targetRunes <= 0 {
		targetRunes = 3000
	}
	segPer := targetRunes / totalSegments
	segMin := segPer - 50
	segMax := segPer + 100
	if segMin < 400 {
		segMin = 400
	}
	return fmt.Sprintf(`你是微电影总导演兼分镜编剧。用户只给「第一镜」文本，你分段输出镜头语言正文（segment_text），%d 段拼接约 %d 汉字，供后续自动切分多镜视频。

%s

只输出 JSON，不要 markdown：
{
  "segment_index": 1,
  "story_background": "仅第1段：一句话故事背景（时代、地点、核心矛盾）",
  "mood": "仅第1段：BGM情绪英文小写 neutral|tense|suspense|sad|warm|hopeful|epic|romantic|horror|comedy",
  "tone": "仅第1段：情绪基调中文",
  "segment_text": "本段正文，汉字 %d-%d 字。按时间顺序写，具体可拍摄，禁止空泛形容词堆砌，禁止重复句凑字数。"
}

分段职责：
- 第1段：从用户第一镜写起（不可偏离核心画面），交代人物外形、服装、场景，铺剧情；segment_text 开头用【铺垫】或等价词标明本段叙事功能；
- 中间段：承接发展，强化镜头语言与表演细节，推进矛盾，须写清场景切换/硬切/推近等过渡；标明【发展】或【转折】；
- 末段：高潮与收束，保持人物外观/服装描述与前面一致（用相同锁定词）；标明【高潮】或【收束】。
每段都必须出现景别、运镜、环境光影中的至少两项，并写清所选镜头如何服务本段剧情节拍；段与段之间须有过渡衔接（如「镜头切换至…」「切至中景…」）。`, totalSegments, targetRunes, BriefCoverageDimensions, segMin, segMax)
}

// OpeningShotBriefSegmentUser 分段请求。
func OpeningShotBriefSegmentUser(opening string, segmentIndex, totalSegments, targetRunes, targetSec int, styleLabel, prevTail string) string {
	if totalSegments <= 0 {
		totalSegments = 3
	}
	if targetRunes <= 0 {
		targetRunes = 3000
	}
	segPer := targetRunes / totalSegments
	segMin := segPer - 50
	segMax := segPer + 100
	if segMin < 400 {
		segMin = 400
	}
	prev := ""
	if segmentIndex > 1 && prevTail != "" {
		prev = fmt.Sprintf("\n\n## 前文末尾（须无缝衔接）\n%s\n", prevTail)
	}
	return fmt.Sprintf(`## 用户第一镜（s01 锚点，第1段必须从此写起）
%s
%s
## 参数
- 本段：第 %d / %d 段（全文目标约 %d 字）
- 成片时长约 %d 秒；画面风格：%s

请输出 JSON，segment_text 必须 %d-%d 汉字，含镜头语言与场景过渡。`, opening, prev, segmentIndex, totalSegments, targetRunes, targetSec, styleLabel, segMin, segMax)
}

// OpeningShotBriefContinueSystem 字数不足时续写。
const OpeningShotBriefContinueSystem = `你是微电影编剧。续写镜头语言文稿 continuation 字段，承接已有内容。

续写每段必须具体可拍摄，至少包含：
1. 景别（wide/medium/close）与机位角度（平视/俯/仰）；
2. 一种主运镜（推/拉/摇/跟/固定）及起止状态；
3. 主光方向、色温、前中后景各至少一项可辨认元素；
4. 表演节拍：预备→动作→收势，含微表情与眼神；
5. 与上一段屏幕方向、服装锁定词一致，不可改写已锁定外形；
6. 场景切换或镜间过渡（硬切/推近/摇镜）须写清。

禁止：重复已有句子；只写抽象情绪（如「气氛紧张」）而无构图；改变人物服装/外形关键词。

` + BriefCoverageDimensions + `

只输出 JSON：{"continuation":"续写 800-1000 汉字，补全尚未写足的剧情/服装/镜头/场景/过渡细节。"}`

// OpeningShotBriefContinueUser 续写请求。
func OpeningShotBriefContinueUser(opening, existingTail string, currentRunes, targetRunes int) string {
	contMin := 600
	contMax := 1000
	if targetRunes < 2500 {
		contMin = 400
		contMax = 700
	}
	return fmt.Sprintf(`## 第一镜锚点
%s

## 当前 %d 字，目标 %d 字

## 文末续写
%s

请输出 JSON continuation，%d-%d 汉字。`, opening, currentRunes, targetRunes, existingTail, contMin, contMax)
}

// ShotsFromBriefSystem 从扩写文稿自动生成分镜。
func ShotsFromBriefSystem(briefRunes int) string {
	if briefRunes <= 0 {
		briefRunes = 3000
	}
	return fmt.Sprintf(`你是分镜师。根据约 %d 字镜头语言文稿，自动切分结构化分镜 JSON，供 AI 绘图/视频渲染。

%s

只输出 JSON：
{
  "shots": [
    {
      "id": "s01",
      "narrative_beat": "本镜剧情功能（铺垫/发展/揭示/高潮/收束等，须与 narration 情绪一致）",
      "brief_excerpt": "扩写原文对应 1-2 句摘录（保证镜头源于文本）",
      "shot_size": "wide|medium|close",
      "camera_angle": "三段式：景别+角度+主运镜（如：中景，平视，缓慢推近）",
      "scene_background": "画面背景与环境",
      "character_motion": "人物移动",
      "dialogue": "对白（无则空字符串）",
      "micro_expression": "微表情",
      "action_behavior": "动作行为",
      "narration": "本镜旁白/TTS（12-45字，本镜独有，禁止与其他镜重复）",
      "visual_prompt": "AI 出图描述（80-180字：须含人物外形+服装锁定词+场景+动作表情+景别光线）",
      "duration_sec": 10,
      "action_beats": ["预备姿态", "动作过程", "收势姿态"],
      "physics_cues": "正向物理：重力+支撑+接触/流体/因果（PhyT2V 七类中选≥2，具体可观察）",
      "forbidden_physics": "反事实负向：与 physics_cues 对立的 3-5 条（穿模/悬浮/瞬移/倒流/因果颠倒等）",
      "character_count": 1,
      "held_props": "本镜道具锁定，格式：右手：匕首；左手：空（双持则两手各写一物；无道具则 右手：空；左手：空）"
    }
  ]
}

要求：
1. s01 严格对应用户第一镜，只可细化不可改写核心画面。
2. 其余镜头覆盖全文剧情，镜数在指定范围内。
3. visual_prompt 各镜重复相同人物外观与服装关键词；每镜写清场景与镜头语言。
4. 至少约三分之一镜头的 scene_background 与上一镜不同（剧情需要时切换场景，可硬切）；同场景镜只保持角色/服装一致，不必复刻上一镜结束画面。
5. 每镜 action_beats 必须 3 条（预备/进行/收势）；narration 不得重复。
6. physics_cues 与 forbidden_physics 必填：cues 写 PhyT2V 正向（重力/支撑/碰撞/因果/流体等）；forbidden 写 PhysVid 反事实负向，须与 cues 对立，各 3-5 条具体短语。
7. held_props 必填格式「右手：X；左手：Y」，与 visual_prompt 一致；单镜内禁止换手（若需换手须拆镜并写放下/接过）。
8. action_beats 每条须重复当前 held_props 的握持手，禁止 beat 间改变持物手或握姿大幅旋转。
9. character_count 默认 1；道具数量与 narration 一致，禁止旁白一把剑、画面两把剑。
10. **物体命名稳定**：同一物体跨镜必须使用相同 canonical 名称（如始终写「匕首」勿混用「短刀」）；跨镜重复出现的重要场景物（王座、信封等）须在 visual_prompt 中保持同一称谓，供 prop-sheets 三视图锁定。
11. 旁白总字数须匹配 target_duration_sec（约 4 字/秒）；过短会导致成片时长远低于目标。
12. narrative_beat 与 brief_excerpt 必填：excerpt 须来自扩写原文；camera_angle 禁止只写单一景别词。
13. visual_prompt 须体现 brief_excerpt 中的场景与动作，镜头语言与剧情内容强关联。`, briefRunes, BriefCoverageDimensions)
}

// ShotsFromBriefUser 扩写文稿 + 第一镜锚点 + 镜数范围。
func ShotsFromBriefUser(opening string, brief string, targetSec int, styleLabel string, minShots, maxShots, briefRunes int) string {
	if briefRunes <= 0 {
		briefRunes = 3000
	}
	if len([]rune(brief)) > 4500 {
		r := []rune(brief)
		brief = string(r[:2200]) + "\n\n……\n\n" + string(r[len(r)-2000:])
	}
	return fmt.Sprintf(`## 用户第一镜（s01 锁定）
%s

## 镜头语言扩写全文（约%d字）
%s

## 参数
- 成片约 %d 秒；风格：%s
- 镜头 %d-%d 镜，每镜 10 秒
- 旁白总字数约 %d 字（约 4 字/秒），各镜 narration 字数之和须接近该值

请输出 JSON（shots 数组）。`, opening, briefRunes, brief, targetSec, styleLabel, minShots, maxShots, targetSec*4)
}
