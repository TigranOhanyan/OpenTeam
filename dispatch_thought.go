package openteam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func (a *Agent) dispatchThought(
	ctx context.Context,
	previousTurnId string,
	logger *zap.Logger,
) (
	orchestrationPlan OrchestrationPlan,
	err error,
) {

	logger = logger.With(zap.String("previousTurnId", previousTurnId))
	logger.Info("getting previous turn...")
	previousTurnRecord, err := a.ConversationHistoryDb.Queries.GetTurn(ctx, previousTurnId)
	logger = logger.With(zap.String("previousTurnId", previousTurnRecord.ID))

	logger.Info("getting duty of previous turn...")
	dutyOfPreviousTurnRecord, err := a.ConversationHistoryDb.Queries.GetDutyByTurn(ctx, previousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get duty", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("dutyId", dutyOfPreviousTurnRecord.ID))

	var llmResponseAsMessage openai.ChatCompletionMessageParamUnion

	var messageId string

	if dutyOfPreviousTurnRecord.StreamMode {
		logger.Info("getting LLM chunk response...")
		llmChunkResponseRecords, er := a.ConversationHistoryDb.Queries.GetLlmChunkResponseByTurn(ctx, previousTurnRecord.ID)
		err = er
		if err != nil {
			logger.Error("failed to get LLM chunk response", zap.Error(err))
			return
		}

		if len(llmChunkResponseRecords) == 0 {
			logger.Error("no chunks in LLM chunk response")
			err = fmt.Errorf("no chunks in LLM chunk response")
			return
		}

		messageId = llmChunkResponseRecords[0].ID

		acc := openai.ChatCompletionAccumulator{}
		for _, record := range llmChunkResponseRecords {
			var chunk openai.ChatCompletionChunk
			err = json.Unmarshal(record.OpenaiChunkResponse, &chunk)
			if err != nil {
				logger.Error("failed to unmarshal llm chunk", zap.Error(err))
				return
			}
			acc.AddChunk(chunk)
		}

		choices := acc.Choices
		amountOfChoices := len(choices)
		if amountOfChoices == 0 {
			logger.Error("no choices in accumulated LLM chunk response")
			err = fmt.Errorf("no choices in accumulated LLM chunk response")
			return
		}

		if amountOfChoices > 1 {
			logger.Error("multiple choices in accumulated LLM chunk response", zap.Int("count", amountOfChoices))
			err = fmt.Errorf("multiple choices in accumulated LLM chunk response")
			return
		}
		choice := choices[0]

		llmResponseAsMessage = choice.Message.ToParam()

	} else {
		logger.Info("getting LLM response...")
		llmResponseRecord, er := a.ConversationHistoryDb.Queries.GetLlmResponseByTurn(ctx, previousTurnRecord.ID)
		err = er
		if err != nil {
			logger.Error("failed to get LLM response", zap.Error(err))
			return
		}

		messageId = llmResponseRecord.ID

		llmResponseJson := llmResponseRecord.OpenaiResponse

		var llmResponse openai.ChatCompletion
		err = json.Unmarshal(llmResponseJson, &llmResponse)
		if err != nil {
			logger.Error("failed to unmarshal llm response", zap.Error(err))
			return
		}

		choices := llmResponse.Choices
		amountOfChoices := len(choices)
		if amountOfChoices == 0 {
			logger.Error("no choices in LLM response")
			err = fmt.Errorf("no choices in LLM response")
			return
		}

		if amountOfChoices > 1 {
			logger.Error("multiple choices in LLM response", zap.Int("count", amountOfChoices))
			err = fmt.Errorf("multiple choices in LLM response")
			return
		}
		choice := choices[0]

		llmResponseAsMessage = choice.Message.ToParam()
	}

	logger = logger.With(zap.String("messageId", messageId))

	memberRecord, err := a.ConversationHistoryDb.Queries.GetMemberByDuty(ctx, dutyOfPreviousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get name", zap.Error(err))
		return
	}
	setName(&llmResponseAsMessage, memberRecord.Name)

	roleOfPreviousTurnRecord, er := a.ConversationHistoryDb.Queries.GetRoleByDuty(ctx, dutyOfPreviousTurnRecord.ID)
	err = er
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}

	logger = logger.With(zap.String("roleId", roleOfPreviousTurnRecord.ID))
	logger.Info("getting channel...")

	channelRecord, er := a.ConversationHistoryDb.Queries.GetChannelByRole(ctx, roleOfPreviousTurnRecord.ID)
	err = er
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}

	completedAt := time.Now().UTC()

	// message decomposition should decompose it into response, handoffs, actions and articulations. I guess some sort of strategy pattern should be implemented to handle different types of decomposition components
	messageStructure, err := deriveMessageStructure(llmResponseAsMessage, logger)
	if err != nil {
		logger.Error("failed to derive message structure", zap.Error(err))
		return
	}

	if messageStructure == nil {

		logger.Info("persisting message...")
		createTurnParams := entities.CreateTurnParams{
			ID:     ulid.Make().String(),
			Kind:   string(EventKindThought),
			Status: string(TurnStatusPending),
		}
		currentTurnRecord, er := a.ConversationHistoryDb.Queries.CreateTurn(ctx, createTurnParams)
		err = er
		if err != nil {
			logger.Error("failed to create current turn", zap.Error(err))
			return
		}

		turnIntoUserMessage(&llmResponseAsMessage)

		llmResponseAsMessageBytes, er := json.Marshal(llmResponseAsMessage)
		err = er
		if err != nil {
			logger.Error("failed to marshal llm response as message", zap.Error(err))
			return
		}

		createMessageParams := entities.CreateMessageParams{
			ID:            messageId,
			Visibility:    string(VisibilityChannel),
			TurnID:        currentTurnRecord.ID,
			ChannelName:   channelRecord.Name,
			RoleID:        roleOfPreviousTurnRecord.ID,
			DutyID:        dutyOfPreviousTurnRecord.ID,
			OpenaiMessage: json.RawMessage(llmResponseAsMessageBytes),
		}

		_, err = a.ConversationHistoryDb.Queries.CreateMessage(ctx, createMessageParams)
		if err != nil {
			logger.Error("failed to create new message", zap.Error(err))
			return
		}

		updateTurnStatusParams := entities.UpdateTurnStatusParams{
			ID:          currentTurnRecord.ID,
			Status:      string(TurnStatusCompleted),
			CompletedAt: sql.NullTime{Time: completedAt, Valid: true},
		}
		_, err = a.ConversationHistoryDb.Queries.UpdateTurnStatus(ctx, updateTurnStatusParams)
		if err != nil {
			logger.Error("failed to update current turn", zap.Error(err))
			return
		}

		orchestrationPlan.replyTurnId = &currentTurnRecord.ID
		return
	}

	for _, articulation := range messageStructure.articulations {

		filteredLlmResponseAsMessage := filteredToolCalls(llmResponseAsMessage, []string{articulation.toolCallId})

		createTurnParams := entities.CreateTurnParams{
			ID:     ulid.Make().String(),
			Kind:   string(EventKindThought),
			Status: string(TurnStatusPending),
		}
		currentTurnRecord, er := a.ConversationHistoryDb.Queries.CreateTurn(ctx, createTurnParams)
		err = er
		if err != nil {
			logger.Error("failed to create current turn", zap.Error(err))
			return
		}

		var fromMemberName string
		fromMemberRecord, er := a.ConversationHistoryDb.Queries.GetMemberByDuty(ctx, dutyOfPreviousTurnRecord.ID)
		err = er
		if err != nil {
			logger.Error("failed to get from member", zap.Error(err))
			return
		}
		fromMemberName = fromMemberRecord.Name

		articulationId := ulid.Make().String()
		createArticulationParams := entities.CreateArticulationParams{
			ID:             articulationId,
			TurnID:         currentTurnRecord.ID,
			ToolCallID:     articulation.toolCallId,
			FromMemberName: fromMemberName,
			ToMemberName:   articulation.toAgent,
			Message:        articulation.message,
		}

		err = a.insertArticulation(ctx, createArticulationParams)
		if err != nil {
			logger.Error("failed to create articulation", zap.Error(err))
			return
		}

		filteredLlmResponseAsMessageBytes, er := json.Marshal(filteredLlmResponseAsMessage)
		err = er
		if err != nil {
			logger.Error("failed to marshal filtered llm response as message", zap.Error(err))
			return
		}

		messageId := ulid.Make().String()

		// TODO: This is a TODO for the future.
		// do we need a hidden message if we have separate tables for articulations, handoffs and actions?
		// since actions are business tools, they will be visible to agents and therefore need a message
		// I think the llm response can have at most one final message but many tool calls,
		// thus the responseId can only be used as a message id for only the final message.
		// maybe we have to explicitly keep a futureMessageId field in the response and chunkResponses tables,
		// even if that will be the same as the id of the respective records.
		// maybe yes, maybe no.
		createMessageParams := entities.CreateMessageParams{
			ID:            messageId,
			Visibility:    string(VisibilityHidden),
			TurnID:        currentTurnRecord.ID,
			ChannelName:   channelRecord.Name,
			RoleID:        roleOfPreviousTurnRecord.ID,
			DutyID:        dutyOfPreviousTurnRecord.ID,
			OpenaiMessage: json.RawMessage(filteredLlmResponseAsMessageBytes),
		}

		_, er = a.ConversationHistoryDb.Queries.CreateMessage(ctx, createMessageParams)
		err = er
		if err != nil {
			logger.Error("failed to create new message", zap.Error(err))
			return
		}

		updateTurnStatusParams := entities.UpdateTurnStatusParams{
			ID:          currentTurnRecord.ID,
			Status:      string(TurnStatusCompleted),
			CompletedAt: sql.NullTime{Time: completedAt, Valid: true},
		}
		_, er = a.ConversationHistoryDb.Queries.UpdateTurnStatus(ctx, updateTurnStatusParams)
		err = er
		if err != nil {
			logger.Error("failed to update current turn", zap.Error(err))
			return
		}

		orchestrationPlan.articulationTurnIds = append(orchestrationPlan.articulationTurnIds, currentTurnRecord.ID)
	}

	return

}

func setName(
	u *openai.ChatCompletionMessageParamUnion,
	name string,
) {

	if vt := u.OfDeveloper; vt != nil && !vt.Name.Valid() {
		vt.Name = param.NewOpt(name)
	} else if vt := u.OfSystem; vt != nil && !vt.Name.Valid() {
		vt.Name = param.NewOpt(name)
	} else if vt := u.OfUser; vt != nil && !vt.Name.Valid() {
		vt.Name = param.NewOpt(name)
	} else if vt := u.OfAssistant; vt != nil && !vt.Name.Valid() {
		vt.Name = param.NewOpt(name)
	}
}

func filteredToolCalls(
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	toolCallIds []string,
) (filteredLlmResponseAsMessage openai.ChatCompletionMessageParamUnion) {
	filteredLlmResponseAsMessage = llmResponseAsMessage
	if param.IsOmitted(llmResponseAsMessage.OfAssistant) {
		return
	}
	filteredToolCalls := []openai.ChatCompletionMessageToolCallUnionParam{}
	for _, toolCall := range llmResponseAsMessage.OfAssistant.ToolCalls {
		if param.IsOmitted(toolCall.OfFunction) {
			continue
		}
		function := toolCall.OfFunction
		if slices.Contains(toolCallIds, function.ID) {
			filteredToolCalls = append(filteredToolCalls, toolCall)
		}
	}
	llmResponseAsMessage.OfAssistant.ToolCalls = filteredToolCalls
	return
}

type thoughtHandoff struct {
	toolCallId string
	toAgent    string
}

type thoughtAction struct {
	toolCallId string
	name       string
	arguments  map[string]interface{}
}

type thoughtArticulation struct {
	toolCallId string
	toAgent    string
	message    string
}

type messageDecomposition struct {
	handoffs      []thoughtHandoff
	actions       []thoughtAction
	articulations []thoughtArticulation
}

func deriveMessageStructure(
	message openai.ChatCompletionMessageParamUnion,
	logger *zap.Logger,
) (decomposition *messageDecomposition, err error) {

	if param.IsOmitted(message.OfAssistant) {
		return
	}

	toolCalls := message.OfAssistant.ToolCalls

	if len(toolCalls) == 0 {
		return
	}

	decomposition = &messageDecomposition{}

	for _, toolCall := range toolCalls {
		if param.IsOmitted(toolCall.OfFunction) {
			continue
		}
		function := toolCall.OfFunction
		var args map[string]interface{}
		err = json.Unmarshal([]byte(function.Function.Arguments), &args)
		if err != nil {
			logger.Error("failed to unmarshal articulattion", zap.Error(err))
			return
		}
		if strings.EqualFold(function.Function.Name, ArticulateToAgentFunction.Name) {
			articulation := thoughtArticulation{
				toolCallId: function.ID,
				toAgent:    args["agent_name"].(string),
				message:    args["message"].(string),
			}
			decomposition.articulations = append(decomposition.articulations, articulation)
		} else if strings.EqualFold(function.Function.Name, handoffToAgentFunction.Name) {
			handoff := thoughtHandoff{
				toolCallId: function.ID,
				toAgent:    args["to_agent"].(string),
			}
			decomposition.handoffs = append(decomposition.handoffs, handoff)
		} else {
			action := thoughtAction{
				toolCallId: function.ID,
				name:       function.Function.Name,
				arguments:  args,
			}
			decomposition.actions = append(decomposition.actions, action)
		}
	}

	return
}

func turnIntoUserMessage(
	message *openai.ChatCompletionMessageParamUnion,
) {

	if param.IsOmitted(message.OfAssistant) {
		return
	}

	assistantContent := message.OfAssistant.Content
	userContent := openai.ChatCompletionUserMessageParamContentUnion{
		OfString: assistantContent.OfString,
	}
	contentParts := []openai.ChatCompletionContentPartUnionParam{}
	for _, assistantPart := range assistantContent.OfArrayOfContentParts {
		if !param.IsOmitted(assistantPart.OfText) {

			textContent := openai.ChatCompletionContentPartUnionParam{
				OfText: assistantPart.OfText,
			}
			contentParts = append(contentParts, textContent)
		}
	}

	if len(contentParts) > 0 {
		userContent.OfArrayOfContentParts = contentParts
	}
	message.OfUser = &openai.ChatCompletionUserMessageParam{
		Content: userContent,
		Name:    message.OfAssistant.Name,
	}
	message.OfAssistant = nil
}
