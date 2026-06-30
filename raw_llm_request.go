package OpenTeam

import (
	"context"
	"encoding/json"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

type rawLLMRequest struct {
	parentExecutionID string
	taskID            string
	chatParams        *openai.ChatCompletionNewParams
}

func (rawLLMRequest rawLLMRequest) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	llmRequestRecord entities.LlmRequest,
	err error,
) {

	rawExecution := rawExecution{
		kind:     "reason",
		parentId: &rawLLMRequest.parentExecutionID,
	}

	executionRecord, er := rawExecution.persist(ctx, qtx, logger)
	err = er
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	chatParamsBytes, err := rawLLMRequest.chatParams.MarshalJSON()
	if err != nil {
		logger.Error("failed to marshal chat params", zap.Error(err))
		return
	}

	createLlmRequestParams := entities.CreateLlmRequestParams{
		ExecutionID:   executionRecord.ID,
		TaskID:        rawLLMRequest.taskID,
		OpenaiRequest: json.RawMessage(chatParamsBytes),
	}
	llmRequestRecord, err = qtx.CreateLlmRequest(ctx, createLlmRequestParams)
	if err != nil {
		logger.Error("failed to create llm request", zap.Error(err))
		return
	}

	return

}
