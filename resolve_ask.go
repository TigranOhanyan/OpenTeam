package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

func (team *Team) Ask(
	ctx context.Context,
	memberName string,
	channelName string,
	message string,
	logger *zap.Logger,
) (
	messageId string,
	err error,
) {
	trx, err := team.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
			return
		}
		err = trx.Commit()
		if err != nil {
			logger.Error("failed to commit transaction", zap.Error(err))
			return
		}
		return
	}()

	qtx := team.ConversationHistoryDb.Queries.WithTx(trx)

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

	rawExecution := rawExecution{
		kind:     "ask",
		parentId: nil,
	}

	executionRecord, err := rawExecution.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	mentions := rawMentions{
		channelName:    channelRecord.Name,
		fromMemberName: memberName,
		fromRoleID:     roleRecord.ID,
		toMemberNames:  allMembersName,
		allMemberNames: allMembersName,
		executionID:    executionRecord.ID,
		message:        message,
	}

	messageRecord, err := mentions.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist mentions and message", zap.Error(err))
		return
	}

	messageId = messageRecord.ID

	return

}
