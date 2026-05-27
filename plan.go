package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
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
	askingStepId string,
	logger *zap.Logger,
) (
	plan Plan,
	err error,
) {

	mentionRecords, err := qtx.GetMentionsBySourceStep(ctx, askingStepId)
	if err != nil {
		logger.Error("failed to get mention", zap.Error(err))
		return
	}

	for _, mentionRecord := range mentionRecords {
		plan.mentionRecords = append(plan.mentionRecords, mentionRecord)
	}

	return
}
