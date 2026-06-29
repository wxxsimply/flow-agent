package config

// LLMRef 返回 stack 中某阶段的 LLM 配置；无 stack 时返回默认 deepseek-v4-flash。
func (app *App) LLMRef(stage string) LLMRef {
	if app != nil && app.Stack != nil {
		if ref, ok := app.Stack.LLM[stage]; ok {
			return ref
		}
	}
	return LLMRef{Provider: "deepseek", Model: "deepseek-v4-flash"}
}
