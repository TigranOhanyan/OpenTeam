package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

type rawExecution struct {
	kind     string
	parentId *string
}

func (execution *rawExecution) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	executionRecord entities.Execution,
	err error,
) {

	createExecutionParams := entities.CreateExecutionParams{
		ID:   ulid.Make().String(),
		Kind: execution.kind,
	}
	executionRecord, err = qtx.CreateExecution(ctx, createExecutionParams)
	if err != nil {
		logger.Error("failed to create current execution", zap.Error(err))
		return
	}

	if execution.parentId != nil {
		createExecutionLinkParams := entities.CreateExecutionLinkParams{
			ParentID: *execution.parentId,
			ChildID:  executionRecord.ID,
		}
		_, err = qtx.CreateExecutionLink(ctx, createExecutionLinkParams)
		if err != nil {
			logger.Error("failed to create execution link", zap.Error(err))
			return
		}
	}

	return
}
