package OpenTeam

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

type rawLLMBulkResponse struct {
	reasonExecutionID string
	llmResponse       *openai.ChatCompletion
}

func (rawLLMBulkResponse rawLLMBulkResponse) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	llmResponseRecord entities.LlmResponse,
	err error,
) {

	if len(rawLLMBulkResponse.llmResponse.Choices) == 0 {
		logger.Error("no choices in llm response")
		err = fmt.Errorf("no choices in llm response")
		return
	}

	logger.Info("LLM called successfully")

	llmResponseBytes, er := json.Marshal(rawLLMBulkResponse.llmResponse)
	err = er
	if err != nil {
		logger.Error("failed to marshal llm response", zap.Error(err))
		return
	}

	createLlmResponseParams := entities.CreateLlmResponseParams{
		ExecutionID: rawLLMBulkResponse.reasonExecutionID,
		Kind:        "bulk",
	}

	_, err = qtx.CreateLlmResponse(ctx, createLlmResponseParams)
	if err != nil {
		logger.Error("failed to create new response", zap.Error(err))
		return
	}

	createLlmBulkResponseParams := entities.CreateLlmBulkResponseParams{
		ExecutionID:    rawLLMBulkResponse.reasonExecutionID,
		OpenaiResponse: json.RawMessage(llmResponseBytes),
	}

	_, err = qtx.CreateLlmBulkResponse(ctx, createLlmBulkResponseParams)
	if err != nil {
		logger.Error("failed to create new bulk response", zap.Error(err))
		return
	}

	_, err = qtx.CloseExecution(ctx, rawLLMBulkResponse.reasonExecutionID)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}

	return

}

type rawLLMChunkResponse struct {
	reasonExecutionID string
	llmResponse       []openai.ChatCompletionChunk
}

func (rawLLMChunkResponse rawLLMChunkResponse) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	llmResponseRecord entities.LlmResponse,
	err error,
) {

	if len(rawLLMChunkResponse.llmResponse) == 0 {
		logger.Error("no chunks in llm response")
		err = fmt.Errorf("no chunks in llm response")
		return
	}

	logger.Info("LLM chunks called successfully")

	createLlmResponseParams := entities.CreateLlmResponseParams{
		ExecutionID: rawLLMChunkResponse.reasonExecutionID,
		Kind:        "chunk",
	}

	_, err = qtx.CreateLlmResponse(ctx, createLlmResponseParams)
	if err != nil {
		logger.Error("failed to create new response", zap.Error(err))
		return
	}

	for sequenceNumber, chunk := range rawLLMChunkResponse.llmResponse {
		createChunkParams := entities.CreateLlmChunkResponsesParams{
			ExecutionID:         rawLLMChunkResponse.reasonExecutionID,
			SequenceNumber:      int64(sequenceNumber),
			OpenaiChunkResponse: json.RawMessage(chunk.RawJSON()),
		}
		err = qtx.CreateLlmChunkResponses(ctx, createChunkParams)
		if err != nil {
			logger.Error("failed to create chunk", zap.Error(err))
			return
		}
	}

	_, err = qtx.CloseExecution(ctx, rawLLMChunkResponse.reasonExecutionID)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}

	return

}
