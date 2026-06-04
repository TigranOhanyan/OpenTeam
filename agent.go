package OpenTeam

import (
	"context"
	"errors"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"go.uber.org/zap"
)

var MultiplePendingStepsError = errors.New("multiple pending steps found")

type agenticReActLoop struct {
	runtime    *AgentRuntime
	runRecord  entities.Run
	stepRecord entities.Step
	channel    entities.Channel
	member     entities.Member
	role       entities.Role
	task       entities.Task
}

func (runtime *AgentRuntime) createAgent(
	ctx context.Context,
	runId string,
	logger *zap.Logger,
) (agent *agenticReActLoop, err error) {

	logger = logger.With(zap.String("runId", runId))
	logger.Info("getting current step...")

	qtx := runtime.ConversationHistoryDb.Queries

	runRecord, err := qtx.GetRun(ctx, runId)
	if err != nil {
		logger.Error("failed to get run", zap.Error(err))
		return
	}

	stepsRecords, err := qtx.GetStepsByRunId(ctx, runRecord.ID)
	if err != nil {
		logger.Error("failed to get steps", zap.Error(err))
		return
	}

	pendingStepRecords := make([]entities.Step, 0)
	for _, stepRecord := range stepsRecords {
		if stepRecord.Status == "pending" {
			pendingStepRecords = append(pendingStepRecords, stepRecord)
		}
	}

	isInconsistent := len(pendingStepRecords) > 1
	if isInconsistent {
		logger.Info("inconsistent steps found", zap.Int("count", len(pendingStepRecords)))
		err = MultiplePendingStepsError
		return
	}

	isCompleted := len(stepsRecords) > 0 && len(pendingStepRecords) == 0

	if isCompleted {
		logger.Info("run is completed", zap.String("runId", runRecord.ID))
		return
	}

	mentionRecord, err := qtx.GetMentionByRun(ctx, runRecord.ID)
	if err != nil {
		logger.Error("failed to get mention", zap.Error(err))
		return
	}

	fromRoleRecord, err := qtx.GetRole(ctx, mentionRecord.FromMemberRoleID)
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

	var pendingStepRecord entities.Step
	if len(pendingStepRecords) == 1 {

		pendingStepRecord = pendingStepRecords[0]

	} else {
		pendingStepRecord, err = createStep(ctx, qtx, runRecord.ID, taskRecord.ID, logger)
		if err != nil {
			logger.Error("failed to create pending step", zap.Error(err))
			return
		}

		if err != nil {
			logger.Error("failed to create pending step", zap.Error(err))
			return
		}

	}

	agent = &agenticReActLoop{
		runtime:    runtime,
		runRecord:  runRecord,
		stepRecord: pendingStepRecord,
		channel:    channelRecord,
		member:     memberRecord,
		role:       roleRecord,
		task:       taskRecord,
	}

	return

}

func (agent *agenticReActLoop) persistMentionsAndMessage(
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

	createMessageParams.StepID = agent.stepRecord.ID
	createMessageParams.TaskID = agent.task.ID

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
			FromMemberRoleID: agent.role.ID,
			ToMemberName:     toMemberRecord.Name,
			Message:          mentions.message,
		}

		mentionRecord, er := qtx.CreateMention(ctx, createMentionParams)
		err = er
		if err != nil {
			logger.Error("failed to create mention", zap.Error(err))
			return
		}

		err = linkRuns(ctx, qtx, agent.runRecord.ID, nextRunRecord.ID, agent.stepRecord.ID, logger)
		if err != nil {
			logger.Error("failed to link run", zap.Error(err))
			return
		}

		if agent.runtime.ChangeStream != nil {

			agent.runtime.ChangeStream <- ChangeEvent{
				Kind:        CdcEventKindMention,
				ChannelName: mentions.channelName,
				MemberName:  toMemberRecord.Name,
				Mention:     &mentionRecord,
			}
		}
	}

	return

}
