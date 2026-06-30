package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type actExecutionResolver struct {
	team *Team
}

func (r *actExecutionResolver) Resolve(
	ctx context.Context,
	actExecution entities.Execution,
	logger *zap.Logger,
) (executionReport ExecutionReport, err error) {

	executionReport.ExecutionID = actExecution.ID

	logger = logger.With(zap.String("actExecutionId", actExecution.ID))
	logger.Info("auto resolving act execution...")

	trx, err := r.team.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := r.team.ConversationHistoryDb.Queries.WithTx(trx)

	allChildren, err := qtx.GetChildExecutions(ctx, actExecution.ID)
	if err != nil {
		logger.Error("failed to get child executions", zap.Error(err))
		return
	}

	if len(allChildren) == 0 {
		logger.Info("act execution has children, skipping...")
		executionReport.Status = ExecutionStatusSkipped
		return
	}

	allChildrenClosed := true
	for _, child := range allChildren {
		if child.Status != "closed" {
			allChildrenClosed = false
			break
		}
	}

	if !allChildrenClosed {
		executionReport.Status = ExecutionStatusSkipped
		return
	}

	_, err = qtx.CloseExecution(ctx, actExecution.ID)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}
	return

}

func (team *Team) AutoResolveActExecution(
	ctx context.Context,
	actExecutionId string,
	logger *zap.Logger,
) (
	err error,
) {

	logger = logger.With(zap.String("actExecutionId", actExecutionId))
	logger.Info("auto resolving act execution...")

	trx, err := team.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := team.ConversationHistoryDb.Queries.WithTx(trx)

	_, err = qtx.CloseExecution(ctx, actExecutionId)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}
	return

}
