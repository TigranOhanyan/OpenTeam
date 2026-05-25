package openteam

import (
	"context"

	"github.com/openteam/entities"
	"go.uber.org/zap"
)

type Plan struct {
	mentionRecords []entities.Mention
	actingStepIds  []string
}

func (o *Plan) isOnlyControl() bool {
	return len(o.mentionRecords) != 0 && len(o.actingStepIds) == 0
}

func (o *Plan) isFinalReply() bool {
	return len(o.mentionRecords) == 0 && len(o.actingStepIds) == 0
}

func reconstitutePlan(
	ctx context.Context,
	qtx *entities.Queries,
	runRecord entities.Run,
	logger *zap.Logger,
) (
	plan Plan,
	err error,
) {

	mentionRecords, err := qtx.GetMentionsByRun(ctx, runRecord.ID)
	if err != nil {
		logger.Error("failed to get mention", zap.Error(err))
		return
	}

	for _, mentionRecord := range mentionRecords {
		plan.mentionRecords = append(plan.mentionRecords, mentionRecord)
	}

	return
}
