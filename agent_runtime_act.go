package OpenTeam

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"go.uber.org/zap"
)

var MissingToolCallIdError = errors.New("missing tool call id")

func (runtime *AgentRuntime) Act(
	ctx context.Context,
	actionRunId string,
	toolResult string,
	logger *zap.Logger,
) (
	toolRequirementMessageId string,
	toolResultMessageId string,
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

	parentRunLinkRecord, err := qtx.GetParentRunLink(ctx, actionRunId)
	if err != nil {
		logger.Error("failed to get parent run link", zap.Error(err))
		return
	}

	spawningStepRecord, err := qtx.GetStep(ctx, parentRunLinkRecord.SpawningStepID)
	if err != nil {
		logger.Error("failed to get spawning step", zap.Error(err))
		return
	}

	actionRecord, err := qtx.GetActionByRun(ctx, actionRunId)
	if err != nil {
		logger.Error("failed to get action", zap.Error(err))
		return
	}

	taskRecord, err := qtx.GetTask(ctx, spawningStepRecord.TaskID)
	if err != nil {
		logger.Error("failed to get task", zap.Error(err))
		return
	}

	roleRecord, err := qtx.GetRole(ctx, taskRecord.RoleID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}

	channelRecord, err := qtx.GetChannel(ctx, roleRecord.ChannelName)
	if err != nil {
		logger.Error("failed to get user room", zap.Error(err))
		return
	}

	// func (r ChatCompletionMessageToolCallUnion) ToParam() ChatCompletionMessageToolCallUnionParam

	messageIdPrefix := ulid.Make().String()
	toolRequirementMessageId = messageIdPrefix + "-0-tool-requirement"
	toolResultMessageId = messageIdPrefix + "-1-tool-result"

	var toolCallOpenAiParam openai.ChatCompletionMessageToolCallUnionParam
	err = json.Unmarshal(actionRecord.ToolCall, &toolCallOpenAiParam)
	if err != nil {
		logger.Error("failed to unmarshal tool call", zap.Error(err))
		return
	}

	toolCallId := toolCallOpenAiParam.GetID()
	if toolCallId == nil {
		logger.Error("tool call id is nil")
		err = MissingToolCallIdError
		return
	}

	toolRequirementOpenAiMessage := openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{toolCallOpenAiParam},
			Name:      param.NewOpt(roleRecord.MemberName),
		},
	}

	toolRequirementOpenAiMessageJson, err := json.Marshal(toolRequirementOpenAiMessage)
	if err != nil {
		logger.Error("failed to marshal tool requirement openai message", zap.Error(err))
		return
	}

	toolRequirementMessageParams := entities.CreateMessageParams{
		ID:            toolRequirementMessageId,
		Visibility:    string(VisibilityTask),
		ChannelName:   channelRecord.Name,
		RoleID:        roleRecord.ID,
		TaskID:        taskRecord.ID,
		OpenaiMessage: json.RawMessage(toolRequirementOpenAiMessageJson),
	}

	toolRequirementMessageRecord, err := qtx.CreateMessage(ctx, toolRequirementMessageParams)
	if err != nil {
		logger.Error("failed to create tool requirement message", zap.Error(err))
		return
	}

	toolResultOpenAiMessage := openai.ChatCompletionMessageParamUnion{
		OfTool: &openai.ChatCompletionToolMessageParam{
			Content: openai.ChatCompletionToolMessageParamContentUnion{
				OfString: param.NewOpt(toolResult),
			},
			ToolCallID: *toolCallId,
		},
	}

	toolResultOpenAiMessageJson, err := json.Marshal(toolResultOpenAiMessage)
	if err != nil {
		logger.Error("failed to marshal tool result openai message", zap.Error(err))
		return
	}

	toolResultMessageParams := entities.CreateMessageParams{
		ID:            toolResultMessageId,
		Visibility:    string(VisibilityTask),
		ChannelName:   channelRecord.Name,
		RoleID:        roleRecord.ID,
		TaskID:        taskRecord.ID,
		OpenaiMessage: json.RawMessage(toolResultOpenAiMessageJson),
	}

	toolResultMessageRecord, err := qtx.CreateMessage(ctx, toolResultMessageParams)
	if err != nil {
		logger.Error("failed to create tool result message", zap.Error(err))
		return
	}

	updateActionMessagesParams := entities.UpdateActionMessagesParams{
		ID:                       actionRecord.ID,
		ToolRequirementMessageID: toolRequirementMessageRecord.ID,
		ToolResultMessageID:      toolResultMessageRecord.ID,
	}

	actionRecord, err = qtx.UpdateActionMessages(ctx, updateActionMessagesParams)
	if err != nil {
		logger.Error("failed to update action", zap.Error(err))
		return
	}

	_, err = qtx.CompleteRun(ctx, actionRecord.RunID)
	if err != nil {
		logger.Error("failed to complete run", zap.Error(err))
		return
	}

	return

}
