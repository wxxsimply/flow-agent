package skills

// Stage 表示 agent 调用 LLM 或拼接 prompt 的阶段。
type Stage string

const (
	StageExpandBriefSegment  Stage = "expand_brief_segment"
	StageExpandBriefContinue Stage = "expand_brief_continue"
	StageGenerateShots       Stage = "generate_shots"
	StageReviewStoryboard    Stage = "review_storyboard"
	StageProduceMotion       Stage = "produce_motion"
)
