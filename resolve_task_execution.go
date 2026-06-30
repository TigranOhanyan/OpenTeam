package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type taskExecutionResolver struct {
	team *Team
}

func (r *taskExecutionResolver) Resolve(
	ctx context.Context,
	taskExecution entities.Execution,
	logger *zap.Logger,
) (executionReport ExecutionReport, err error) {

	executionReport.ExecutionID = taskExecution.ID

	logger = logger.With(zap.String("taskExecutionId", taskExecution.ID))
	logger.Info("resolving task execution...")

	trx, err := r.team.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
			return
		}
		err = trx.Commit()
		if err != nil {
			logger.Error("failed to commit transaction", zap.Error(err))
			return
		}
		return
	}()

	qtx := r.team.ConversationHistoryDb.Queries.WithTx(trx)

	taskRecord, err := qtx.GetTask(ctx, taskExecution.ID)
	if err != nil {
		logger.Error("failed to get task", zap.Error(err))
		return
	}

	allChildren, err := qtx.GetChildExecutions(ctx, taskExecution.ID)
	if err != nil {
		logger.Error("failed to get child executions", zap.Error(err))
		return
	}

	if len(allChildren) == 0 {
		logger.Info("task execution has children, creating react loop...")
		executionReport.Status = ExecutionStatusExpanded
		rawReactLoop := rawReactLoop{
			taskExecutionId: taskExecution.ID,
			taskID:          taskRecord.ID,
		}
		_, err = rawReactLoop.persist(ctx, qtx, logger)
		if err != nil {
			logger.Error("failed to persist react loop", zap.Error(err))
			return
		}

		return
	}

	latestChild := allChildren[0]
	if latestChild.Status != "closed" {
		executionReport.Status = ExecutionStatusSkipped
		return
	}

	latestChildReActRecord, err := qtx.GetReactLoop(ctx, latestChild.ID)
	if err != nil {
		logger.Error("failed to get react loop", zap.Error(err))
		return
	}

	if latestChildReActRecord.Status == "pending" {
		executionReport.Status = ExecutionStatusSkipped
		return
	}

	if latestChildReActRecord.Status == "reason" {
		executionReport.Status = ExecutionStatusClosed
		_, err = qtx.UpdateReactLoopStatus(ctx, entities.UpdateReactLoopStatusParams{
			Status:      "reason",
			ExecutionID: latestChildReActRecord.ExecutionID,
		})
		if err != nil {
			logger.Error("failed to update react loop status", zap.Error(err))
			return
		}
		return
	}

	if latestChildReActRecord.Status == "act" {
		executionReport.Status = ExecutionStatusExpanded
		rawReactLoop := rawReactLoop{
			taskExecutionId: taskExecution.ID,
			taskID:          taskRecord.ID,
		}
		_, err = rawReactLoop.persist(ctx, qtx, logger)
		if err != nil {
			logger.Error("failed to persist react loop", zap.Error(err))
			return
		}
		return
	}

	return
}
