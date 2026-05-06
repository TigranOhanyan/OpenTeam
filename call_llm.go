package openteam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func (a *Agent) callLLM(
	ctx context.Context,
	previousTurnId string,
	logger *zap.Logger,
) (
	currentTurnId string,
	err error,
) {

	logger = logger.With(zap.String("previousTurnId", previousTurnId))
	logger.Info("getting previous turn...")

	previousTurnRecord, err := a.ConversationHistoryDb.Queries.GetTurn(ctx, previousTurnId)
	if err != nil {
		logger.Error("failed to get turn", zap.Error(err))
		return
	}

	logger.Info("getting duty...")

	dutyRecord, err := a.ConversationHistoryDb.Queries.GetDutyByTurn(ctx, previousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get persona", zap.Error(err))
		return
	}

	memberRecord, err := a.ConversationHistoryDb.Queries.GetMemberByDuty(ctx, dutyRecord.ID)
	if err != nil {
		logger.Error("failed to get name", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("name", memberRecord.Name))

	currentTurnRecord, err := a.ConversationHistoryDb.Queries.CreateTurn(ctx, entities.CreateTurnParams{
		ID:     ulid.Make().String(),
		Kind:   string(EventKindThinking),
		Status: string(TurnStatusPending),
	})
	if err != nil {
		logger.Error("failed to create current turn", zap.Error(err))
		return
	}

	logger = logger.With(zap.String("currentTurnId", currentTurnRecord.ID))
	logger.Info("getting role...")

	roleRecord, err := a.ConversationHistoryDb.Queries.GetRoleByDuty(ctx, dutyRecord.ID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}

	logger = logger.With(zap.String("roleId", roleRecord.ID))
	logger.Info("getting channel...")

	channelRecord, err := a.ConversationHistoryDb.Queries.GetChannelByRole(ctx, roleRecord.ID)
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}

	logger = logger.With(zap.String("channelId", channelRecord.Name))
	logger.Info("getting messages...")

	getContextMessagesParams := entities.GetContextMessagesParams{
		DutyID:      dutyRecord.ID,
		RoleID:      roleRecord.ID,
		ChannelName: channelRecord.Name,
	}
	messageRecords, err := a.ConversationHistoryDb.Queries.GetContextMessages(ctx, getContextMessagesParams)
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
		turnIntoAssistantMessage(&openAiMessage, memberRecord.Name)
		openAiMessages[i] = openAiMessage
	}

	chatParams := openai.ChatCompletionNewParams{
		Model:       dutyRecord.Model,
		Messages:    openAiMessages,
		N:           param.NewOpt(AmountOfChoices),
		Temperature: param.NewOpt(Temperature),
		// ParallelToolCalls: param.NewOpt(ParallelToolCalls),
	}

	responseId := ulid.Make().String()
	logger = logger.With(zap.String("responseId", responseId))

	if dutyRecord.StreamMode {
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
				TurnID:              currentTurnRecord.ID,
				DutyID:              dutyRecord.ID,
				OpenaiChunkResponse: json.RawMessage(chunk.RawJSON()),
			}
			err = a.insertChunk(ctx, createChunkParams)
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
		// 5. Mark the turn as completed
		now := time.Now().UTC()
		updateTurnStatusParams := entities.UpdateTurnStatusParams{
			ID:          currentTurnRecord.ID,
			Status:      string(TurnStatusCompleted),
			CompletedAt: sql.NullTime{Time: now, Valid: true},
		}
		_, err = a.ConversationHistoryDb.Queries.UpdateTurnStatus(ctx, updateTurnStatusParams)
		if err != nil {
			logger.Error("failed to update current turn", zap.Error(err))
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

		now := time.Now().UTC()
		updateTurnStatusParams := entities.UpdateTurnStatusParams{
			ID:          currentTurnRecord.ID,
			Status:      string(TurnStatusCompleted),
			CompletedAt: sql.NullTime{Time: now, Valid: true},
		}

		_, err = a.ConversationHistoryDb.Queries.UpdateTurnStatus(ctx, updateTurnStatusParams)
		if err != nil {
			logger.Error("failed to update current turn", zap.Error(err))
			return
		}

		llmResponseBytes, er := json.Marshal(llmResponse)
		err = er
		if err != nil {
			logger.Error("failed to marshal llm response", zap.Error(err))
			return
		}

		createLlmResponseParams := entities.CreateLlmResponseParams{
			ID:             responseId,
			DutyID:         dutyRecord.ID,
			TurnID:         currentTurnRecord.ID,
			OpenaiResponse: json.RawMessage(llmResponseBytes),
		}

		_, err = a.ConversationHistoryDb.Queries.CreateLlmResponse(ctx, createLlmResponseParams)
		if err != nil {
			logger.Error("failed to create new response", zap.Error(err))
			return
		}

	}

	currentTurnId = currentTurnRecord.ID

	return
}

func turnIntoAssistantMessage(
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
}
