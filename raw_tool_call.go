package OpenTeam

import (
	"context"
	"encoding/json"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

type rawToolCallExecution struct {
	parentExecutionID string
	toolCall          openai.ChatCompletionMessageToolCallUnionParam
	llmResponseID     string
}

func (rawToolCallExecution rawToolCallExecution) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	actionExecutionRecord entities.Action,
	err error,
) {

	rawExecution := rawExecution{
		kind:     "action",
		parentId: &rawToolCallExecution.parentExecutionID,
	}

	executionRecord, er := rawExecution.persist(ctx, qtx, logger)
	err = er
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	toolCallBytes, er := rawToolCallExecution.toolCall.MarshalJSON()
	err = er
	if err != nil {
		logger.Error("failed to marshal tool call", zap.Error(err))
		return
	}

	createActionParams := entities.CreateActionParams{
		ID:            ulid.Make().String(),
		ExecutionID:   executionRecord.ID,
		ToolCall:      json.RawMessage(toolCallBytes),
		LlmResponseID: rawToolCallExecution.llmResponseID,
	}

	actionExecutionRecord, er = qtx.CreateAction(ctx, createActionParams)
	err = er
	if err != nil {
		logger.Error("failed to create mention", zap.Error(err))
		return
	}

	return

}
