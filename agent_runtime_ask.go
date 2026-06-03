package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

func (runtime *AgentRuntime) Ask(
	ctx context.Context,
	memberName string,
	channelName string,
	message string,
	logger *zap.Logger,
) (
	messageId string,
	err error,
) {
	defer func() {
		if runtime.ChangeStream != nil {
			close(runtime.ChangeStream)
		}
	}()

	trx, err := runtime.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}
	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := runtime.ConversationHistoryDb.Queries.WithTx(trx)

	channelRecord, err := qtx.GetChannel(ctx, channelName)
	if err != nil {
		logger.Error("failed to get user room", zap.Error(err))
		return
	}

	fromRoleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  memberName,
		ChannelName: channelRecord.Name,
	}

	roleRecord, err := qtx.GetRoleByMemberAndChannel(ctx, fromRoleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get user role", zap.Error(err))
		return
	}

	allRoleRecords, err := qtx.GetRoleByChannel(ctx, channelRecord.Name)
	if err != nil {
		logger.Error("failed to get user membership", zap.Error(err))
		return
	}

	allMembersName := make([]string, len(allRoleRecords))
	for i, roleRecord := range allRoleRecords {
		allMembersName[i] = roleRecord.MemberName
	}

	mention := mentions{
		channelName:    channelRecord.Name,
		fromMemberName: memberName,
		fromRoleID:     roleRecord.ID,
		toMemberNames:  allMembersName,
		allMemberNames: allMembersName,
		message:        message,
	}

	messageRecord, err := runtime.persistMentionsAndMessage(ctx, qtx, mention, logger)
	if err != nil {
		logger.Error("failed to persist mentions", zap.Error(err))
		return
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	messageId = messageRecord.ID

	return

}
