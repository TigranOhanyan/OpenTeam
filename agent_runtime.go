package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

type AgentRuntime struct {
	ConversationHistoryDb *TeamDb
	LlmClient             *openai.Client
}

func (runtime *AgentRuntime) persistMentionsAndMessage(
	ctx context.Context,
	qtx *entities.Queries,
	mentions rawMentions,
	logger *zap.Logger,
) (
	messageRecord entities.Message,
	cdcEvents []CdcEvent,
	err error,
) {

	createMessageParams, err := mentions.createMessageParams(ctx, logger)
	if err != nil {
		logger.Error("failed to create message params", zap.Error(err))
		return
	}

	messageRecord, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create message", zap.Error(err))
		return
	}

	for _, roleRecord := range mentions.toMemberNames {
		if roleRecord == mentions.fromMemberName {
			continue
		}

		toMemberRecord, er := qtx.GetMember(ctx, roleRecord)
		err = er
		if err != nil {
			logger.Error("failed to get agent member", zap.Error(err))
			return
		}

		nextRunRecord, er := createRun(ctx, qtx, "mention", logger)
		err = er
		if err != nil {
			logger.Error("failed to create run", zap.Error(err))
			return
		}

		createMentionParams := entities.CreateMentionParams{
			ID:               ulid.Make().String(),
			RunID:            nextRunRecord.ID,
			MessageID:        messageRecord.ID,
			FromMemberRoleID: mentions.fromRoleID,
			ToMemberName:     toMemberRecord.Name,
			Message:          mentions.message,
		}

		mentionRecord, er := qtx.CreateMention(ctx, createMentionParams)
		err = er
		if err != nil {
			logger.Error("failed to create mention", zap.Error(err))
			return
		}

		event := CdcEvent{
			Kind:        CdcEventKindMention,
			ChannelName: mentions.channelName,
			MemberName:  toMemberRecord.Name,
			Mention:     &mentionRecord,
		}

		cdcEvents = append(cdcEvents, event)

	}

	return

}
