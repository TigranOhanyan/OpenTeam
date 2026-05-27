package openteam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

type AgentRuntime struct {
	ConversationHistoryDb *TeamDb
	LlmClient             *openai.Client
	ChangeStream          chan<- ChangeEvent
}

func (runtime *AgentRuntime) createAgent(
	ctx context.Context,
	mentionRecord entities.Mention,
	logger *zap.Logger,
) (a agent, err error) {

	logger = logger.With(zap.String("mentionId", mentionRecord.ID))
	logger.Info("getting current step...")

	qtx := runtime.ConversationHistoryDb.Queries

	runRecord, err := qtx.GetRun(ctx, mentionRecord.RunID)
	if err != nil {
		logger.Error("failed to get run", zap.Error(err))
		return
	}

	fromRoleRecord, err := qtx.GetRoleByTask(ctx, mentionRecord.FromMemberTaskID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("roleId", fromRoleRecord.ID))

	channelRecord, err := qtx.GetChannelByRole(ctx, fromRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("channelId", channelRecord.Name))

	memberRecord, err := qtx.GetMember(ctx, mentionRecord.ToMemberName)
	if err != nil {
		logger.Error("failed to get to participant", zap.Error(err))
		return
	}

	roleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  memberRecord.Name,
		ChannelName: channelRecord.Name,
	}

	roleRecord, err := qtx.GetRoleByMemberAndChannel(ctx, roleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get to membership", zap.Error(err))
		return
	}

	taskRecord, err := qtx.GetFirstTask(ctx, roleRecord.ID)
	if err != nil {
		logger.Error("failed to get to persona", zap.Error(err))
		return
	}

	a = agent{
		runtime:   runtime,
		runRecord: runRecord,
		channel:   channelRecord,
		member:    memberRecord,
		role:      roleRecord,
		task:      taskRecord,
	}

	return

}

type agent struct {
	runtime   *AgentRuntime
	runRecord entities.Run
	channel   entities.Channel
	member    entities.Member
	role      entities.Role
	task      entities.Task
}
