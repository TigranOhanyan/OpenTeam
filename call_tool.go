package openteam

import (
	"context"

	"go.uber.org/zap"
)

func (runtime *AgentRuntime) CallTool(
	ctx context.Context,
	previousStepId string,
	logger *zap.Logger,
) (
	nextStepId string,
	err error,
) {
	return
}
