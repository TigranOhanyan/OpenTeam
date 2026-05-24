package openteam

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func linkSteps(
	ctx context.Context,
	qtx *entities.Queries,
	prevStepId string,
	nextStepId string,
	logger *zap.Logger,
) (
	err error,
) {
	linkedAt := time.Now().UTC()
	createStepLinkParams := entities.CreateStepLinkParams{
		PrevID:   prevStepId,
		NextID:   nextStepId,
		LinkedAt: sql.NullTime{Time: linkedAt, Valid: true},
	}
	_, err = qtx.CreateStepLink(ctx, createStepLinkParams)
	if err != nil {
		logger.Error("failed to create step link", zap.Error(err))
		return
	}

	return
}

func createStep(
	ctx context.Context,
	qtx *entities.Queries,
	stepKind EventKind,
	logger *zap.Logger,
) (
	stepRecord entities.Step,
	err error,
) {
	stepRecordParams := entities.CreateStepParams{
		ID:   ulid.Make().String(),
		Kind: string(stepKind),
	}
	stepRecord, err = qtx.CreateStep(ctx, stepRecordParams)
	if err != nil {
		logger.Error("failed to create current step", zap.Error(err))
		return
	}

	return
}
