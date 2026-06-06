package OpenTeam

import (
	"context"
	"fmt"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

type AgentRuntime struct {
	ConversationHistoryDb *TeamDb
	LlmClient             *openai.Client
	ChangeStream          chan<- ChangeEvent
}

func (runtime *AgentRuntime) persistMentionsAndMessage(
	ctx context.Context,
	qtx *entities.Queries,
	mentions mentions,
	logger *zap.Logger,
) (
	messageRecord entities.Message,
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

		createRunParams := entities.CreateRunParams{
			ID:   ulid.Make().String(),
			Kind: "mention",
		}
		nextRunRecord, er := qtx.CreateRun(ctx, createRunParams)
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

		fmt.Println("nextRunRecord", nextRunRecord) // todo

		if runtime.ChangeStream != nil {

			event := ChangeEvent{
				Kind:        CdcEventKindMention,
				ChannelName: mentions.channelName,
				MemberName:  toMemberRecord.Name,
				Mention:     &mentionRecord,
			}

			runtime.ChangeStream <- event
		}
	}

	return

}
