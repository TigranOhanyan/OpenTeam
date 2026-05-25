package openteam

import (
	"context"

	"go.uber.org/zap"
)

func (agent *agent) reAct(
	ctx context.Context,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {
	logger.Info("getting current step...")

	qtx := agent.runtime.ConversationHistoryDb.Queries

	observeStepRecord, err := createStep(ctx, qtx, EventKindObserving, &agent.runRecord.ID, logger)
	if err != nil {
		logger.Error("failed to create init step", zap.Error(err))
		return
	}

	reply, err = agent.reason(ctx, observeStepRecord, logger)
	if err != nil {
		logger.Error("failed to reason", zap.Error(err))
		return
	}

	return
}
