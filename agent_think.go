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

func (a *Agent) think(
	ctx context.Context,
	stepRecord entities.Step,
	mention Mention,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {
	logger = logger.With(zap.String("stepId", stepRecord.ID))

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

	planningStepRecord, err := createStep(ctx, qtx, EventKindPlanning, logger)
	if err != nil {
		logger.Error("failed to create planning step", zap.Error(err))
		return
	}

	err = linkSteps(ctx, qtx, stepRecord.ID, planningStepRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link steps", zap.Error(err))
		return
	}

	logger = logger.With(zap.String("channelId", mention.channelName))
	logger.Info("getting messages...")

	getContextMessagesParams := entities.GetContextMessagesParams{
		TaskID:      mention.toTask.ID,
		RoleID:      mention.toRoleID,
		ChannelName: mention.channelName,
	}
	messageRecords, err := qtx.GetContextMessages(ctx, getContextMessagesParams)
	if err != nil {
		logger.Error("failed to get messages", zap.Error(err))
		return
	}

	amountOfMessages := len(messageRecords)
	logger.Info("relevant messages loaded", zap.Int("count", amountOfMessages))

	openAiMessages := make([]openai.ChatCompletionMessageParamUnion, amountOfMessages)
	for i, messageRecord := range messageRecords {
		openAiMessageJson := messageRecord.OpenaiMessage
		var openAiMessage openai.ChatCompletionMessageParamUnion
		err = json.Unmarshal(openAiMessageJson, &openAiMessage)
		if err != nil {
			logger.Error("failed to unmarshal openai message", zap.Error(err))
			return
		}
		stepIntoAssistantMessage(&openAiMessage, mention.toMemberName)
		openAiMessages[i] = openAiMessage
	}

	chatParams := openai.ChatCompletionNewParams{
		Model:       mention.toTask.Model,
		Messages:    openAiMessages,
		N:           param.NewOpt(AmountOfChoices),
		Temperature: param.NewOpt(Temperature),
		// ParallelToolCalls: param.NewOpt(ParallelToolCalls),
	}

	responseId := ulid.Make().String()
	logger = logger.With(zap.String("responseId", responseId))

	if mention.toTask.StreamMode {
		logger.Info("calling LLM in stream mode...")
		// 1. Start the stream
		stream := a.LlmClient.Chat.Completions.NewStreaming(ctx, chatParams)
		defer stream.Close()
		// 2. Generate a single ID for the entire stream
		sequenceNumber := 0

		// 3. Iterate over the stream
		for stream.Next() {

			logger := logger.With(zap.Int("sequenceNumber", sequenceNumber))

			err = stream.Err()
			if err != nil {
				logger.Error("stream error", zap.Error(err))
				return
			}

			chunk := stream.Current()

			logger.Info("processing chunk")

			// Persist the raw chunk to the database immediately
			createChunkParams := entities.CreateLlmChunkResponsesParams{
				ID:                  responseId,
				SequenceNumber:      int64(sequenceNumber),
				StepID:              planningStepRecord.ID,
				TaskID:              mention.toTask.ID,
				OpenaiChunkResponse: json.RawMessage(chunk.RawJSON()),
			}
			err = a.insertChunk(ctx, qtx, createChunkParams, logger)
			if err != nil {
				logger.Error("failed to insert chunk", zap.Error(err))
				return
			}
			sequenceNumber++
		}
		// 4. Check for stream errors
		if err = stream.Err(); err != nil {
			logger.Error("stream error", zap.Error(err))
			return
		}

	} else {

		logger.Info("calling LLM in one shot...")

		maybeLlmResponse, er := a.LlmClient.Chat.Completions.New(ctx, chatParams)
		err = er
		if err != nil {
			logger.Error("failed to call LLM", zap.Error(err))
			return
		}

		if maybeLlmResponse == nil {
			logger.Error("no llm response received")
			err = fmt.Errorf("no llm response received")
			return
		}

		llmResponse := *maybeLlmResponse

		if len(llmResponse.Choices) == 0 {
			logger.Error("no choices in llm response")
			err = fmt.Errorf("no choices in llm response")
			return
		}

		logger.Info("LLM called successfully")

		llmResponseBytes, er := json.Marshal(llmResponse)
		err = er
		if err != nil {
			logger.Error("failed to marshal llm response", zap.Error(err))
			return
		}

		createLlmResponseParams := entities.CreateLlmResponseParams{
			ID:             responseId,
			TaskID:         mention.toTask.ID,
			StepID:         planningStepRecord.ID,
			OpenaiResponse: json.RawMessage(llmResponseBytes),
		}

		_, err = qtx.CreateLlmResponse(ctx, createLlmResponseParams)
		if err != nil {
			logger.Error("failed to create new response", zap.Error(err))
			return
		}

	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	reply, err = a.analyze(ctx, planningStepRecord, mention, logger)
	if err != nil {
		logger.Error("failed to think", zap.Error(err))
		return
	}

	return
}

func stepIntoAssistantMessage(
	message *openai.ChatCompletionMessageParamUnion,
	name string,
) {

	if param.IsOmitted(message.OfUser) {
		return
	}

	if param.IsOmitted(message.OfUser.Name) {
		return
	}

	if message.OfUser.Name.Value != name {
		return
	}

	userContent := message.OfUser.Content
	assistantContent := openai.ChatCompletionAssistantMessageParamContentUnion{
		OfString: userContent.OfString,
	}
	assistantContentParts := []openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{}
	for _, userContentPart := range userContent.OfArrayOfContentParts {
		if !param.IsOmitted(userContentPart.OfText) {

			textContent := openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
				OfText: userContentPart.OfText,
			}
			assistantContentParts = append(assistantContentParts, textContent)
		}
	}

	if len(assistantContentParts) > 0 {
		assistantContent.OfArrayOfContentParts = assistantContentParts
	}
	message.OfAssistant = &openai.ChatCompletionAssistantMessageParam{
		Content: assistantContent,
		Name:    message.OfUser.Name,
	}
	message.OfUser = nil

	return
}

type ReplyOrThinkOver struct {
	reply      *Reply
	mentionIds []string
}
