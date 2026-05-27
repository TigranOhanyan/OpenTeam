package OpenTeam

import (
	"context"

	"go.uber.org/zap"
)

type Runtime interface {
	Ask(ctx context.Context, memberName string, channelName string, message string, logger *zap.Logger) (runIds []string, err error)
	Run(ctx context.Context, runId string, logger *zap.Logger) (reply Reply, err error)
}
