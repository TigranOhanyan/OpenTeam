package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type rawReactLoop struct {
	taskExecutionId string
	taskID          string
}

func (rawReactLoop rawReactLoop) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	reactLoopRecord entities.ReactLoop,
	err error,
) {

	rawExecution := rawExecution{
		kind:     "react",
		parentId: &rawReactLoop.taskExecutionId,
	}

	executionRecord, er := rawExecution.persist(ctx, qtx, logger)
	err = er
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	createReactLoopParams := entities.CreateReactLoopParams{
		ExecutionID: executionRecord.ID,
		TaskID:      rawReactLoop.taskID,
	}

	reactLoopRecord, er = qtx.CreateReactLoop(ctx, createReactLoopParams)
	err = er
	if err != nil {
		logger.Error("failed to create react loop", zap.Error(err))
		return
	}

	return

}
