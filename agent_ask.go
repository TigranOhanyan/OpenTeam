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
	CdcEventKindAddressing CdcEventKind = "addressing"
)

type ChangeEvent struct {
	Kind        CdcEventKind
	TurnID      string
	ChannelName string
	Action      *entities.Action
	Chunk       *entities.LlmChunkResponse
	Addressing  *entities.Addressing
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

	fromDutyRecord, err := qtx.GetFirstDuty(ctx, fromRoleRecord.ID)
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

		addressingTurnRecord, er := createTurn(ctx, qtx, EventKindAddressing, logger)
		err = er
		if err != nil {
			logger.Error("failed to create addressing turn", zap.Error(err))
			return
		}

		toolCallId := ulid.Make().String()
		createAddressingParams := entities.CreateAddressingParams{
			ID:               ulid.Make().String(),
			TurnID:           addressingTurnRecord.ID,
			ToolCallID:       toolCallId,
			FromMemberDutyID: fromDutyRecord.ID,
			ToMemberName:     toMemberRecord.Name,
			Message:          message,
		}

		_, err = qtx.CreateAddressing(ctx, createAddressingParams)
		if err != nil {
			return
		}
		orchestrationPlan.addressingTurnIds = append(orchestrationPlan.addressingTurnIds, addressingTurnRecord.ID)
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	for _, addressingTurnId := range orchestrationPlan.addressingTurnIds {
		addressingReply, er := a.reply(ctx, addressingTurnId, logger) // short circuit if the reply requries tool calls
		err = er
		if err != nil {
			logger.Error("failed to reply to addressing", zap.Error(err))
			return
		}
		reply.actionIds = append(reply.actionIds, addressingReply.actionIds...)
		reply.replyTurnIds = append(reply.replyTurnIds, addressingReply.replyTurnIds...)
	}

	return

}
