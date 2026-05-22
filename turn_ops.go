package openteam

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func linkTurns(
	ctx context.Context,
	qtx *entities.Queries,
	prevTurnId string,
	nextTurnId string,
	logger *zap.Logger,
) (
	err error,
) {
	linkedAt := time.Now().UTC()
	createTurnLinkParams := entities.CreateTurnLinkParams{
		PrevID:   prevTurnId,
		NextID:   nextTurnId,
		LinkedAt: sql.NullTime{Time: linkedAt, Valid: true},
	}
	_, err = qtx.CreateTurnLink(ctx, createTurnLinkParams)
	if err != nil {
		logger.Error("failed to create turn link", zap.Error(err))
		return
	}

	return
}

func createTurn(
	ctx context.Context,
	qtx *entities.Queries,
	turnKind EventKind,
	logger *zap.Logger,
) (
	turnRecord entities.Turn,
	err error,
) {
	turnRecordParams := entities.CreateTurnParams{
		ID:   ulid.Make().String(),
		Kind: string(turnKind),
	}
	turnRecord, err = qtx.CreateTurn(ctx, turnRecordParams)
	if err != nil {
		logger.Error("failed to create current turn", zap.Error(err))
		return
	}

	return
}
