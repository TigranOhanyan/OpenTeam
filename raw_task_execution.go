package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type rawTaskExecution struct {
	parentExecutionID string
	taskID            string
}

func (rawTaskExecution rawTaskExecution) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	taskExecutionRecord entities.TaskExecution,
	err error,
) {

	rawExecution := rawExecution{
		kind:     "task",
		parentId: &rawTaskExecution.parentExecutionID,
	}

	executionRecord, er := rawExecution.persist(ctx, qtx, logger)
	err = er
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	createTaskExecutionParams := entities.CreateTaskExecutionParams{
		TaskID:      rawTaskExecution.taskID,
		ExecutionID: executionRecord.ID,
	}

	taskExecutionRecord, er = qtx.CreateTaskExecution(ctx, createTaskExecutionParams)
	err = er
	if err != nil {
		logger.Error("failed to create mention", zap.Error(err))
		return
	}

	return

}
