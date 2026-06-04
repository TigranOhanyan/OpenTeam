package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

func createStep(
	ctx context.Context,
	qtx *entities.Queries,
	runId string,
	taskId string,
	logger *zap.Logger,
) (
	stepRecord entities.Step,
	err error,
) {
	stepRecordParams := entities.CreateStepParams{
		ID:     ulid.Make().String(),
		RunID:  runId,
		TaskID: taskId,
	}

	stepRecord, err = qtx.CreateStep(ctx, stepRecordParams)
	if err != nil {
		logger.Error("failed to create current step", zap.Error(err))
		return
	}

	return
}
