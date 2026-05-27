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
	askingStepId string,
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

	taskRecord, err := qtx.GetFirstTask(ctx, roleRecord.ID)
	if err != nil {
		logger.Error("failed to get user persona", zap.Error(err))
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

	askingStepRecord, er := createStep(ctx, qtx, EventKindAsking, nil, logger)
	err = er
	if err != nil {
		logger.Error("failed to create mention step", zap.Error(err))
		return
	}

	mention := mention{
		channelName:    channelRecord.Name,
		fromMemberName: memberName,
		fromRoleID:     roleRecord.ID,
		fromTaskID:     taskRecord.ID,
		toMemberNames:  allMembersName,
		allMemberNames: allMembersName,
		message:        message,
	}

	_, err = mention.persistMessage(ctx, qtx, askingStepRecord, logger)
	if err != nil {
		logger.Error("failed to persist message", zap.Error(err))
		return
	}

	_, err = runtime.persistMentions(ctx, qtx, mention, askingStepRecord, logger)
	if err != nil {
		logger.Error("failed to persist mentions", zap.Error(err))
		return
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	askingStepId = askingStepRecord.ID

	return

}
