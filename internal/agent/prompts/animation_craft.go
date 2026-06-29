package prompts



// AnimationCraftPerformance 人类直觉：表演与镜头可读性（兜底，与 .cursor/skills 同步）。

const AnimationCraftPerformance = `

表演与镜头（人类直觉）：

- 单镜结构：预备→动作→反应→收势；action_beats 三步须可画、幅度小。

- 情绪用眼神、呼吸、肩线、重心表达，禁止空泛形容词。

- 运镜每镜一种主运动（推/拉/摇/跟/固定），与人物移动方向一致。

- 景别有变化或延续理由；写按时间推进的视觉事件，非静态海报。

- 跨镜：屏幕方向、服装锁定词、光线色调一致。`



// AnimationCraftVisualPolish 画面精致度兜底。

const AnimationCraftVisualPolish = `

画面精致度：

- 写明主光方向、色温、景深；肤色自然不过曝。

- 每镜至少一种材质细节（织物、皮肤、环境）。

- 禁止「电影感」「精致」等空词；竖屏 9:16 主体勿贴底字幕区。`



// AnimationCraftPhysicsPrompt 物理现实兜底。

const AnimationCraftPhysicsPrompt = `

物理现实（每镜必填，PhyT2V 正向 + PhysVid 反事实负向）：

- 单镜单一主动作；action_beats 三步中仅中间一步含主动作，首尾为预备/收势。

- physics_cues：重力/支撑/碰撞/因果/流体/材质/光热，至少两类，可观察具体描述。

- forbidden_physics：与 cues 对立的反事实 3-5 条（穿模、悬浮、瞬移、倒流、因果颠倒、数量突变）。

- 单镜单一主动作；关节合理；物体碰撞有反应；有液体时写流向与液面变化。`



// StoryboardReviewSystem LLM 分镜审查兜底（8 维）。

const StoryboardReviewSystem = `你是动画审片编辑。按 8 维审查分镜 JSON：故事、s01锚定、staging、节奏、表演、连续性、物理、画面精致度、旁白去重。

输出 patches 修复 visual_prompt/narration/physics_cues/forbidden_physics/action_beats。

只输出 JSON：{"patches":[{"shot_id":"s01","physics_cues":"..."}],"summary":"..."}`


