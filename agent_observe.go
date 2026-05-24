package openteam

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func (a *Agent) observe(
	ctx context.Context,
	stepID string,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {
	logger = logger.With(zap.String("currentStepId", stepID))
	logger.Info("getting current step...")

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

	stepRecord, err := qtx.GetStep(ctx, stepID)
	if err != nil {
		logger.Error("failed to get step", zap.Error(err))
		return
	}

	mention, err := reconstituteMention(ctx, qtx, stepRecord, logger)
	if err != nil {
		logger.Error("failed to reconstitute observing context", zap.Error(err))
		return
	}

	observingStepRecord, err := mention.persistMentionMessage(ctx, qtx, stepRecord, logger)
	if err != nil {
		logger.Error("failed to persist mention message", zap.Error(err))
		return
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	reply, err = a.reason(ctx, observingStepRecord, mention, logger)
	if err != nil {
		logger.Error("failed to reason", zap.Error(err))
		return
	}

	return
}

type Mention struct {
	mentionID      string
	stepID         string
	channelName    string
	fromMemberName string
	fromRoleID     string
	fromTaskID     string
	toMemberName   string
	toRoleID       string
	toTask         entities.Task
	message        string
}

func (a Mention) persistMentionMessage(
	ctx context.Context,
	qtx *entities.Queries,
	stepRecord entities.Step,
	logger *zap.Logger,
) (
	observingStepRecord entities.Step,
	err error,
) {

	observingStepRecord, err = createStep(ctx, qtx, EventKindObserving, logger)
	if err != nil {
		logger.Error("failed to create observing step", zap.Error(err))
		return
	}

	err = linkSteps(ctx, qtx, stepRecord.ID, observingStepRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link steps", zap.Error(err))
		return
	}

	mentionMessage := fmt.Sprintf("%s! %s", a.toMemberName, a.message)

	openAiMessage := openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: param.NewOpt(mentionMessage),
			},
			Name: param.NewOpt(a.fromMemberName),
		},
	}

	openAiMessageBytes, err := json.Marshal(openAiMessage)
	if err != nil {
		logger.Error("failed to marshal openai message", zap.Error(err))
		return
	}

	createMessageParams := entities.CreateMessageParams{
		ID:            ulid.Make().String(),
		OpenaiMessage: json.RawMessage(openAiMessageBytes),
		Visibility:    string(VisibilityChannel),
		StepID:        observingStepRecord.ID,
		ChannelName:   a.channelName,
		RoleID:        a.toRoleID,
		TaskID:        a.toTask.ID,
	}
	_, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create new message", zap.Error(err))
		return
	}

	return
}

func reconstituteMention(
	ctx context.Context,
	qtx *entities.Queries,
	stepRecord entities.Step,
	logger *zap.Logger,
) (
	mention Mention,
	err error,
) {

	logger = logger.With(zap.String("stepId", stepRecord.ID))

	mentionRecord, err := qtx.GetMentionByStep(ctx, stepRecord.ID)
	if err != nil {
		logger.Error("failed to get mention", zap.Error(err))
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

	fromMemberRecord, err := qtx.GetMemberByTask(ctx, mentionRecord.FromMemberTaskID)
	if err != nil {
		logger.Error("failed to get from member", zap.Error(err))
		return
	}

	toMemberRecord, err := qtx.GetMember(ctx, mentionRecord.ToMemberName)
	if err != nil {
		logger.Error("failed to get to participant", zap.Error(err))
		return
	}

	toRoleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  toMemberRecord.Name,
		ChannelName: channelRecord.Name,
	}

	toRoleRecord, err := qtx.GetRoleByMemberAndChannel(ctx, toRoleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get to membership", zap.Error(err))
		return
	}

	toTaskRecord, err := qtx.GetFirstTask(ctx, toRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get to persona", zap.Error(err))
		return
	}

	mention = Mention{
		mentionID:      mentionRecord.ID,
		stepID:         stepRecord.ID,
		channelName:    channelRecord.Name,
		fromMemberName: fromMemberRecord.Name,
		fromRoleID:     fromRoleRecord.ID,
		fromTaskID:     mentionRecord.FromMemberTaskID,
		toMemberName:   toMemberRecord.Name,
		toRoleID:       toRoleRecord.ID,
		toTask:         toTaskRecord,
		message:        mentionRecord.Message,
	}

	return
}

type Reply struct {
	replyStepIds []string
	actionIds    []string
}
