package openteam

import (
	"context"

	"go.uber.org/zap"
)

func (a *Agent) CallTool(
	ctx context.Context,
	previousStepId string,
	logger *zap.Logger,
) (
	nextStepId string,
	err error,
) {
	return
}
