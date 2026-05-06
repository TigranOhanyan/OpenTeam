package openteam

import (
	"context"

	"go.uber.org/zap"
)

func (a *Agent) CallTool(
	ctx context.Context,
	previousTurnId string,
	logger *zap.Logger,
) (
	nextTurnId string,
	err error,
) {
	return
}
