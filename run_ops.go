package openteam

import (
	"context"

	"github.com/oklog/ulid/v2"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func createRun(
	ctx context.Context,
	qtx *entities.Queries,
	sourceStepId string,
	logger *zap.Logger,
) (
	runRecord entities.Run,
	err error,
) {
	createRunParams := entities.CreateRunParams{
		ID:           ulid.Make().String(),
		SourceStepID: sourceStepId,
	}
	runRecord, err = qtx.CreateRun(ctx, createRunParams)
	if err != nil {
		logger.Error("failed to create current run", zap.Error(err))
		return
	}

	return
}
