package openteam

import (
	"context"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

type CdcEventKind string

const (
	CdcEventKindAction     CdcEventKind = "action"
	CdcEventKindChunk      CdcEventKind = "chunk"
	CdcEventKindMentioning CdcEventKind = "mentioning"
)

type ChangeEvent struct {
	Kind        CdcEventKind
	StepID      string
	ChannelName string
	Action      *entities.Action
	Chunk       *entities.LlmChunkResponse
	Mention     *entities.Mention
}

type Agent struct {
	ConversationHistoryDb *TeamDb
	LlmClient             *openai.Client
	ChangeStream          chan<- ChangeEvent
}

func (a *Agent) Ask(
	ctx context.Context,
	memberName string,
	channelName string,
	message string,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {
	defer func() {
		if a.ChangeStream != nil {
			close(a.ChangeStream)
		}
	}()

	trx, err := a.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}
	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := a.ConversationHistoryDb.Queries.WithTx(trx)

	channelRecord, err := qtx.GetChannel(ctx, channelName)
	if err != nil {
		logger.Error("failed to get user room", zap.Error(err))
		return
	}

	fromRoleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  memberName,
		ChannelName: channelRecord.Name,
	}

	fromRoleRecord, err := qtx.GetRoleByMemberAndChannel(ctx, fromRoleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get user role", zap.Error(err))
		return
	}

	fromTaskRecord, err := qtx.GetFirstTask(ctx, fromRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get user persona", zap.Error(err))
		return
	}

	allRoleRecords, err := qtx.GetRoleByChannel(ctx, channelRecord.Name)
	if err != nil {
		logger.Error("failed to get user membership", zap.Error(err))
		return
	}

	orchestrationPlan := Plan{}

	for _, roleRecord := range allRoleRecords {
		if roleRecord.ID == fromRoleRecord.ID {
			continue
		}

		toMemberRecord, er := qtx.GetMember(ctx, roleRecord.MemberName)
		err = er
		if err != nil {
			logger.Error("failed to get agent member", zap.Error(err))
			return
		}

		mentionStepRecord, er := createStep(ctx, qtx, EventKindMentioning, logger)
		err = er
		if err != nil {
			logger.Error("failed to create mention step", zap.Error(err))
			return
		}

		toolCallId := ulid.Make().String()
		createMentionParams := entities.CreateMentionParams{
			ID:               ulid.Make().String(),
			StepID:           mentionStepRecord.ID,
			ToolCallID:       toolCallId,
			FromMemberTaskID: fromTaskRecord.ID,
			ToMemberName:     toMemberRecord.Name,
			Message:          message,
		}

		_, err = qtx.CreateMention(ctx, createMentionParams)
		if err != nil {
			return
		}
		orchestrationPlan.mentionStepIds = append(orchestrationPlan.mentionStepIds, mentionStepRecord.ID)
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	for _, mentionStepId := range orchestrationPlan.mentionStepIds {
		mentionReply, er := a.observe(ctx, mentionStepId, logger) // short circuit if the reply requries tool calls
		err = er
		if err != nil {
			logger.Error("failed to observe mention", zap.Error(err))
			return
		}
		reply.actionIds = append(reply.actionIds, mentionReply.actionIds...)
		reply.replyStepIds = append(reply.replyStepIds, mentionReply.replyStepIds...)
	}

	return

}
