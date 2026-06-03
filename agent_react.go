package OpenTeam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"go.uber.org/zap"
)

var UnexpectedMessageStructureError = errors.New("unexpected message structure")
var InvalidMentionArgumentsError = errors.New("invalid mention arguments")

func (agent *agenticReActLoop) reAct(
	ctx context.Context,
	stepRecord entities.Step,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {
	logger = logger.With(zap.String("stepId", stepRecord.ID))

	logger = logger.With(zap.String("channelId", agent.channel.Name))
	logger.Info("getting messages...")

	getContextMessagesParams := entities.GetContextMessagesParams{
		TaskID:      agent.task.ID,
		RoleID:      agent.role.ID,
		ChannelName: agent.channel.Name,
	}
	messageRecords, err := agent.runtime.ConversationHistoryDb.Queries.GetContextMessages(ctx, getContextMessagesParams)
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
		turnIntoAssistantMessage(&openAiMessage, agent.member.Name)
		openAiMessages[i] = openAiMessage
	}

	chatParams := openai.ChatCompletionNewParams{
		Model:       agent.task.Model,
		Messages:    openAiMessages,
		N:           param.NewOpt(AmountOfChoices),
		Temperature: param.NewOpt(Temperature),
		// ParallelToolCalls: param.NewOpt(ParallelToolCalls),
	}

	trx, err := agent.runtime.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := agent.runtime.ConversationHistoryDb.Queries.WithTx(trx)

	var llmResponseAsMessage openai.ChatCompletionMessageParamUnion
	var messageId string

	if agent.task.StreamMode {
		llmResponseAsMessage, messageId, err = agent.reasonInStreamMode(ctx, qtx, stepRecord, chatParams, logger)
		if err != nil {
			logger.Error("failed to reason in stream mode", zap.Error(err))
			return
		}
	} else {
		llmResponseAsMessage, messageId, err = agent.reasonInOneShotMode(ctx, qtx, stepRecord, chatParams, logger)
		if err != nil {
			logger.Error("failed to reason in one shot mode", zap.Error(err))
			return
		}
	}

	logger = logger.With(zap.String("messageId", messageId))

	if param.IsOmitted(llmResponseAsMessage.OfAssistant) {
		err = UnexpectedMessageStructureError
		return
	}

	assistantMessage := llmResponseAsMessage.OfAssistant

	orchestrationPlan := Plan{}

	for _, toolCall := range assistantMessage.ToolCalls {
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
		if strings.EqualFold(function.Function.Name, mentionMemberFunction.Name) {

			mentionOrchestrationPlan, er := agent.persistMention(ctx, qtx, args, function.ID, stepRecord, logger)
			err = er
			if err != nil {
				logger.Error("failed to persist mention", zap.Error(err))
				return
			}

			orchestrationPlan.mentionRecords = append(orchestrationPlan.mentionRecords, mentionOrchestrationPlan.mentionRecords...)
			orchestrationPlan.actingStepIds = append(orchestrationPlan.actingStepIds, mentionOrchestrationPlan.actingStepIds...)
		} else {
			err = UnexpectedMessageStructureError
			return
		}
	}

	if !orchestrationPlan.isOnlyControl() {
		logger.Info("persisting message...")

		setName(&llmResponseAsMessage, agent.member.Name)

		turnIntoUserMessage(&llmResponseAsMessage)

		filteredLlmResponseAsMessage := filterControlToolCalls(llmResponseAsMessage)

		filteredLlmResponseAsMessageBytes, er := json.Marshal(filteredLlmResponseAsMessage)
		err = er
		if err != nil {
			logger.Error("failed to marshal llm response as message", zap.Error(err))
			return
		}

		createMessageParams := entities.CreateMessageParams{
			ID:            messageId,
			Visibility:    string(VisibilityChannel),
			StepID:        stepRecord.ID,
			ChannelName:   agent.channel.Name,
			RoleID:        agent.role.ID,
			TaskID:        agent.task.ID,
			OpenaiMessage: json.RawMessage(filteredLlmResponseAsMessageBytes),
		}

		_, err = qtx.CreateMessage(ctx, createMessageParams)
		if err != nil {
			logger.Error("failed to create new message", zap.Error(err))
			return
		}

		reply.ReplyStepIds = append(reply.ReplyStepIds, stepRecord.ID)
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	return
}

func (agent *agenticReActLoop) reasonInStreamMode(
	ctx context.Context,
	qtx *entities.Queries,
	stepRecord entities.Step,
	chatParams openai.ChatCompletionNewParams,
	logger *zap.Logger,
) (
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	messageId string,
	err error,
) {

	responseId := ulid.Make().String()
	messageId = responseId
	logger = logger.With(zap.String("responseId", responseId))

	logger.Info("calling LLM in stream mode...")
	// 1. Start the stream
	stream := agent.runtime.LlmClient.Chat.Completions.NewStreaming(ctx, chatParams)
	defer stream.Close()

	// 3. Iterate over the stream
	sequenceNumber := 0
	acc := openai.ChatCompletionAccumulator{}

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
			StepID:              stepRecord.ID,
			TaskID:              agent.task.ID,
			OpenaiChunkResponse: json.RawMessage(chunk.RawJSON()),
		}

		_, err = qtx.CreateLlmChunkResponses(ctx, createChunkParams)
		if err != nil {
			logger.Error("failed to create chunk", zap.Error(err))
			return
		}

		if agent.runtime.ChangeStream != nil {

			agent.runtime.ChangeStream <- ChangeEvent{
				Kind:        CdcEventKindMessageChunk,
				MemberName:  agent.member.Name,
				ChannelName: agent.channel.Name,
				Chunk: &entities.LlmChunkResponse{
					RunID:               agent.runRecord.ID,
					ID:                  createChunkParams.ID,
					SequenceNumber:      createChunkParams.SequenceNumber,
					StepID:              createChunkParams.StepID,
					TaskID:              createChunkParams.TaskID,
					OpenaiChunkResponse: createChunkParams.OpenaiChunkResponse,
				},
			}
		}

		sequenceNumber++
		acc.AddChunk(chunk)
	}
	// 4. Check for stream errors
	if err = stream.Err(); err != nil {
		logger.Error("stream error", zap.Error(err))
		return
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

	return

}

func (agent *agenticReActLoop) reasonInOneShotMode(
	ctx context.Context,
	qtx *entities.Queries,
	stepRecord entities.Step,
	chatParams openai.ChatCompletionNewParams,
	logger *zap.Logger,
) (
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	messageId string,
	err error,
) {

	responseId := ulid.Make().String()
	messageId = responseId
	logger = logger.With(zap.String("responseId", responseId))
	logger.Info("calling LLM in one shot...")

	maybeLlmResponse, er := agent.runtime.LlmClient.Chat.Completions.New(ctx, chatParams)
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
		TaskID:         agent.task.ID,
		StepID:         stepRecord.ID,
		OpenaiResponse: json.RawMessage(llmResponseBytes),
	}

	_, err = qtx.CreateLlmResponse(ctx, createLlmResponseParams)
	if err != nil {
		logger.Error("failed to create new response", zap.Error(err))
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

	return
}

func (runtime *AgentRuntime) llmResponseToMessage(
	ctx context.Context,
	stepId string,
	task entities.Task,
	logger *zap.Logger,
) (
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	messageId string,
	err error,
) {

	if task.StreamMode {
		logger.Info("getting LLM chunk response...")
		llmChunkResponseRecords, er := runtime.ConversationHistoryDb.Queries.GetLlmChunkResponseByStep(ctx, stepId)
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
		llmResponseRecord, er := runtime.ConversationHistoryDb.Queries.GetLlmResponseByStep(ctx, stepId)
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
	return
}

func (agent *agenticReActLoop) persistMention(
	ctx context.Context,
	qtx *entities.Queries,
	arguments map[string]interface{},
	toolCallId string,
	sourceStep entities.Step,
	logger *zap.Logger,
) (
	orchestrationPlan Plan,
	err error,
) {

	toMemberName := arguments["agent_name"].(string)
	if toMemberName == "" {
		err = InvalidMentionArgumentsError
		return
	}
	message := arguments["message"].(string)
	if message == "" {
		err = InvalidMentionArgumentsError
		return
	}

	allRoleRecords, err := qtx.GetRoleByChannel(ctx, agent.channel.Name)
	if err != nil {
		logger.Error("failed to get user membership", zap.Error(err))
		return
	}

	allMembersName := make([]string, len(allRoleRecords))
	for i, roleRecord := range allRoleRecords {
		allMembersName[i] = roleRecord.MemberName
	}

	mentions := mentions{
		channelName:    agent.channel.Name,
		fromMemberName: agent.member.Name,
		fromRoleID:     agent.role.ID,
		toMemberNames:  []string{toMemberName},
		allMemberNames: []string{toMemberName},
		message:        message,
	}

	orchestrationPlan, err = agent.persistMentionsAndMessage(ctx, qtx, mentions, logger)
	if err != nil {
		logger.Error("failed to persist mention", zap.Error(err))
		return
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
