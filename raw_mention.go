package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type rawMention struct {
	parentExecutionID string
	messageID         string
	fromRoleID        string
	toMemberName      string
	message           string
}

func (rawMention rawMention) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	mentionRecord entities.Mention,
	err error,
) {

	toMemberRecord, er := qtx.GetMember(ctx, rawMention.toMemberName)
	err = er
	if err != nil {
		logger.Error("failed to get agent member", zap.Error(err))
		return
	}

	rawExecution := rawExecution{
		kind:     "mention",
		parentId: &rawMention.parentExecutionID,
	}

	executionRecord, er := rawExecution.persist(ctx, qtx, logger)
	err = er
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	createMentionParams := entities.CreateMentionParams{
		ExecutionID:      executionRecord.ID,
		MessageID:        rawMention.messageID,
		FromMemberRoleID: rawMention.fromRoleID,
		ToMemberName:     toMemberRecord.Name,
		Message:          rawMention.message,
	}

	mentionRecord, er = qtx.CreateMention(ctx, createMentionParams)
	err = er
	if err != nil {
		logger.Error("failed to create mention", zap.Error(err))
		return
	}

	return

}
