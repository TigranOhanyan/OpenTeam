package OpenTeam

import (
	"context"

	"github.com/openai/openai-go/v3"
	"go.uber.org/zap"
)

func (team *Team) ResolveLLMCallExecutionBulk(
	ctx context.Context,
	reasonExecutionId string,
	llmResponse *openai.ChatCompletion,
	logger *zap.Logger,
) (
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
		}
	}()

	qtx := team.ConversationHistoryDb.Queries.WithTx(trx)

	llmPartialRequestRecord, err := qtx.GetPartialLlmRequestByExecution(ctx, reasonExecutionId)
	if err != nil {
		logger.Error("failed to get llm partial request", zap.Error(err))
		return
	}

	rawLLMBulkResponse := rawLLMBulkResponse{
		reasonExecutionID: reasonExecutionId,
		llmRequestId:      llmPartialRequestRecord.ID,
		llmResponse:       llmResponse,
	}

	_, err = rawLLMBulkResponse.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist llm response", zap.Error(err))
		return
	}

	_, err = qtx.CloseExecution(ctx, reasonExecutionId)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}

	return

}

func (team *Team) ResolveLLMCallExecutionChunk(
	ctx context.Context,
	reasonExecutionId string,
	llmResponse []openai.ChatCompletionChunk,
	logger *zap.Logger,
) (
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
		}
	}()

	qtx := team.ConversationHistoryDb.Queries.WithTx(trx)

	llmPartialRequestRecord, err := qtx.GetPartialLlmRequestByExecution(ctx, reasonExecutionId)
	if err != nil {
		logger.Error("failed to get llm partial request", zap.Error(err))
		return
	}

	rawLLMChunkResponse := rawLLMChunkResponse{
		reasonExecutionID: reasonExecutionId,
		llmRequestId:      llmPartialRequestRecord.ID,
		llmResponse:       llmResponse,
	}

	_, err = rawLLMChunkResponse.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist llm response", zap.Error(err))
		return
	}

	_, err = qtx.CloseExecution(ctx, reasonExecutionId)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}

	return

}
