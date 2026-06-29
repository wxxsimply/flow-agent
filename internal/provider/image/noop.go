package image

import (
	"context"
	"fmt"
)

// Noop 未配置出图时使用。
type Noop struct{}

func (Noop) Generate(ctx context.Context, req GenerateRequest) ([]byte, error) {
	_ = ctx
	_ = req
	return nil, fmt.Errorf("image: not configured")
}
