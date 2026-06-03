package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func createRun(
	ctx context.Context,
	qtx *entities.Queries,
	mentionId string,
	logger *zap.Logger,
) (
	runRecord entities.Run,
	err error,
) {
	createRunParams := entities.CreateRunParams{
		ID:        ulid.Make().String(),
		MentionID: mentionId,
	}
	runRecord, err = qtx.CreateRun(ctx, createRunParams)
	if err != nil {
		logger.Error("failed to create current run", zap.Error(err))
		return
	}

	return
}

func linkRuns(
	ctx context.Context,
	qtx *entities.Queries,
	parentRunId string,
	childRunId string,
	spawningStepId string,
	logger *zap.Logger,
) (
	err error,
) {
	createRunLinkParams := entities.CreateRunLinkParams{
		ParentRunID:    parentRunId,
		ChildRunID:     childRunId,
		SpawningStepID: spawningStepId,
	}
	_, err = qtx.CreateRunLink(ctx, createRunLinkParams)
	if err != nil {
		logger.Error("failed to create run link", zap.Error(err))
		return
	}

	return
}
