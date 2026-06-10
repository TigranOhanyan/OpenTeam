package OpenTeam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	logger *zap.Logger,
) (
	err error,
) {

	err = agent.reActUnsafe(ctx, logger)
	if err == nil {
		return
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

	outOfOfficeMessage := agent.outOfOffice()
	outOfOfficeMessageBytes, er := json.Marshal(outOfOfficeMessage)
	err = er
	if err != nil {
		logger.Error("failed to marshal out of office message", zap.Error(err))
		return
	}

	createMessageParams := entities.CreateMessageParams{
		ID:            ulid.Make().String(),
		Visibility:    string(VisibilityChannel),
		StepID:        agent.stepRecord.ID,
		ChannelName:   agent.channelRecord.Name,
		RoleID:        agent.roleRecord.ID,
		TaskID:        agent.taskRecord.ID,
		OpenaiMessage: json.RawMessage(outOfOfficeMessageBytes),
	}

	_, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create new message", zap.Error(err))
		return
	}

	_, err = qtx.CompleteRun(ctx, agent.runRecord.ID)
	if err != nil {
		logger.Error("failed to update run", zap.Error(err))
		return
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	return
}

func (agent *agenticReActLoop) reActUnsafe(
	ctx context.Context,
	logger *zap.Logger,
) (
	err error,
) {
	logger = logger.With(zap.String("stepId", agent.stepRecord.ID))

	logger = logger.With(zap.String("channelId", agent.channelRecord.Name))
	logger.Info("getting messages...")

	getContextMessagesParams := entities.GetContextMessagesParams{
		TaskID:      agent.taskRecord.ID,
		RoleID:      agent.roleRecord.ID,
		ChannelName: agent.channelRecord.Name,
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
		openAiMessage = turnIntoAssistantMessage(openAiMessage, agent.memberRecord.Name)
		openAiMessages[i] = openAiMessage
	}

	toolRecords, err := agent.runtime.ConversationHistoryDb.Queries.GetToolsByTask(ctx, agent.taskRecord.ID)
	if err != nil {
		logger.Error("failed to get tools", zap.Error(err))
		return
	}

	openAiTools := make([]openai.ChatCompletionToolUnionParam, len(toolRecords))
	for i, toolRecord := range toolRecords {
		openAiToolJson := toolRecord.Tool
		var openAiTool openai.ChatCompletionToolUnionParam
		err = json.Unmarshal(openAiToolJson, &openAiTool)
		if err != nil {
			logger.Error("failed to unmarshal openai tool", zap.Error(err))
			return
		}
		openAiTools[i] = openAiTool
	}

	chatParams := openai.ChatCompletionNewParams{
		Model:       agent.taskRecord.Model,
		Messages:    openAiMessages,
		N:           param.NewOpt(AmountOfChoices),
		Temperature: param.NewOpt(Temperature),
	}

	if len(openAiTools) > 0 {
		chatParams.Tools = openAiTools
		chatParams.ParallelToolCalls = param.NewOpt(ParallelToolCalls)
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
	messageId := ulid.Make().String()
	logger = logger.With(zap.String("messageId", messageId))

	chatParamsBytes, err := chatParams.MarshalJSON()
	if err != nil {
		logger.Error("failed to marshal chat params", zap.Error(err))
		return
	}

	createLlmRequestParams := entities.CreateLlmRequestParams{
		ID:            messageId,
		TaskID:        agent.taskRecord.ID,
		StepID:        agent.stepRecord.ID,
		OpenaiRequest: json.RawMessage(chatParamsBytes),
	}
	_, err = qtx.CreateLlmRequest(ctx, createLlmRequestParams)
	if err != nil {
		logger.Error("failed to create llm request", zap.Error(err))
		return
	}

	if agent.taskRecord.StreamMode {
		llmResponseAsMessage, err = agent.reasonInStreamMode(ctx, qtx, messageId, agent.stepRecord, chatParams, logger)
		if err != nil {
			logger.Error("failed to reason in stream mode", zap.Error(err))
			return
		}
	} else {
		llmResponseAsMessage, err = agent.reasonInOneShotMode(ctx, qtx, messageId, agent.stepRecord, chatParams, logger)
		if err != nil {
			logger.Error("failed to reason in one shot mode", zap.Error(err))
			return
		}
	}

	setName(&llmResponseAsMessage, agent.memberRecord.Name)

	logger = logger.With(zap.String("messageId", messageId))

	if param.IsOmitted(llmResponseAsMessage.OfAssistant) {
		err = UnexpectedMessageStructureError
		return
	}

	orchestrationPlan := plan{}
	err = orchestrationPlan.parse(llmResponseAsMessage)
	if err != nil {
		logger.Error("failed to parse plan", zap.Error(err))
		return
	}

	for _, mentionPlan := range orchestrationPlan.mentionsPlans {
		_, err = agent.persistMention(ctx, qtx, mentionPlan, logger)
		if err != nil {
			logger.Error("failed to persist mention", zap.Error(err))
			return
		}
	}

	for _, actionPlan := range orchestrationPlan.actionPlans {

		_, err = agent.persistAction(ctx, qtx, actionPlan, logger)
		if err != nil {
			logger.Error("failed to persist action", zap.Error(err))
			return
		}

	}

	if orchestrationPlan.hasMessage {
		logger.Info("persisting message...")

		userLlmResponseAsMessage := turnIntoUserMessage(llmResponseAsMessage)

		userLlmResponseAsMessageBytes, er := json.Marshal(userLlmResponseAsMessage)
		err = er
		if err != nil {
			logger.Error("failed to marshal llm response as message", zap.Error(err))
			return
		}

		createMessageParams := entities.CreateMessageParams{
			ID:            messageId,
			Visibility:    string(VisibilityChannel),
			StepID:        agent.stepRecord.ID,
			ChannelName:   agent.channelRecord.Name,
			RoleID:        agent.roleRecord.ID,
			TaskID:        agent.taskRecord.ID,
			OpenaiMessage: json.RawMessage(userLlmResponseAsMessageBytes),
		}

		_, err = qtx.CreateMessage(ctx, createMessageParams)
		if err != nil {
			logger.Error("failed to create new message", zap.Error(err))
			return
		}

	}

	logger.Info("marking step as complete...")
	_, err = qtx.CompleteStep(ctx, agent.stepRecord.ID)
	if err != nil {
		logger.Error("failed to update step", zap.Error(err))
		return
	}

	if !orchestrationPlan.isFinalReply() {
		logger.Info("creating new step...")
		createStepParams := entities.CreateStepParams{
			ID:     ulid.Make().String(),
			RunID:  agent.runRecord.ID,
			TaskID: agent.taskRecord.ID,
		}
		_, err = qtx.CreateStep(ctx, createStepParams)
		if err != nil {
			logger.Error("failed to create new step", zap.Error(err))
			return
		}
	} else {
		logger.Info("marking run as complete...")
		_, err = qtx.CompleteRun(ctx, agent.runRecord.ID)
		if err != nil {
			logger.Error("failed to update run", zap.Error(err))
			return
		}
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
	messageId string,
	stepRecord entities.Step,
	chatParams openai.ChatCompletionNewParams,
	logger *zap.Logger,
) (
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	err error,
) {

	responseId := messageId

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
			TaskID:              agent.taskRecord.ID,
			OpenaiChunkResponse: json.RawMessage(chunk.RawJSON()),
		}

		chunkRecord, er := qtx.CreateLlmChunkResponses(ctx, createChunkParams)
		err = er
		if err != nil {
			logger.Error("failed to create chunk", zap.Error(err))
			return
		}

		if agent.runtime.ChangeStream != nil {

			event := ChangeEvent{
				Kind:        CdcEventKindMessageChunk,
				MemberName:  agent.memberRecord.Name,
				ChannelName: agent.channelRecord.Name,
				Chunk:       &chunkRecord,
			}

			agent.runtime.ChangeStream <- event
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
	messageId string,
	stepRecord entities.Step,
	chatParams openai.ChatCompletionNewParams,
	logger *zap.Logger,
) (
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	err error,
) {

	responseId := messageId
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
		TaskID:         agent.taskRecord.ID,
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
	message openai.ChatCompletionMessageParamUnion,
	name string,
) openai.ChatCompletionMessageParamUnion {

	if param.IsOmitted(message.OfUser) {
		return message
	}

	if param.IsOmitted(message.OfUser.Name) {
		return message
	}

	if message.OfUser.Name.Value != name {
		return message
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

	return openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			Content: assistantContent,
			Name:    message.OfUser.Name,
		},
	}
}

func (agent *agenticReActLoop) persistMention(
	ctx context.Context,
	qtx *entities.Queries,
	mentionPlan mentionPlan,
	logger *zap.Logger,
) (
	messageRecord entities.Message,
	err error,
) {
	allRoleRecords, err := qtx.GetRoleByChannel(ctx, agent.channelRecord.Name)
	if err != nil {
		logger.Error("failed to get user membership", zap.Error(err))
		return
	}

	allMembersName := make([]string, len(allRoleRecords))
	for i, roleRecord := range allRoleRecords {
		allMembersName[i] = roleRecord.MemberName
	}

	mentions := mentions{
		channelName:    agent.channelRecord.Name,
		fromMemberName: agent.memberRecord.Name,
		fromRoleID:     agent.roleRecord.ID,
		toMemberNames:  []string{mentionPlan.agentName},
		allMemberNames: []string{mentionPlan.agentName},
		message:        mentionPlan.message,
	}

	messageRecord, err = agent.persistMentionsAndMessage(ctx, qtx, mentions, logger)
	if err != nil {
		logger.Error("failed to persist mention", zap.Error(err))
		return
	}

	return

}

func (agent *agenticReActLoop) persistAction(
	ctx context.Context,
	qtx *entities.Queries,
	actionPlan actionPlan,
	logger *zap.Logger,
) (
	actionRecord entities.Action,
	err error,
) {
	nextRunRecord, er := createRun(ctx, qtx, "action", logger)
	err = er
	if err != nil {
		logger.Error("failed to create run", zap.Error(err))
		return
	}

	toolCallBytes, er := json.Marshal(actionPlan.toolCall)
	err = er
	if err != nil {
		logger.Error("failed to marshal tool call", zap.Error(err))
		return
	}

	createActionParams := entities.CreateActionParams{
		ID:       ulid.Make().String(),
		RunID:    nextRunRecord.ID,
		ToolCall: json.RawMessage(toolCallBytes),
	}

	actionRecord, err = qtx.CreateAction(ctx, createActionParams)
	err = er
	if err != nil {
		logger.Error("failed to create mention", zap.Error(err))
		return
	}

	err = linkRuns(ctx, qtx, agent.runRecord.ID, nextRunRecord.ID, agent.stepRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link run", zap.Error(err))
		return
	}

	if agent.runtime.ChangeStream != nil {

		event := ChangeEvent{
			Kind:        CdcEventKindAction,
			ChannelName: agent.channelRecord.Name,
			MemberName:  agent.memberRecord.Name,
			Action:      &actionRecord,
		}

		agent.runtime.ChangeStream <- event
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
	message openai.ChatCompletionMessageParamUnion,
) openai.ChatCompletionMessageParamUnion {

	if param.IsOmitted(message.OfAssistant) {
		return message
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

	return openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: userContent,
			Name:    message.OfAssistant.Name,
		},
	}
}

func (agent *agenticReActLoop) outOfOffice() openai.ChatCompletionMessageParamUnion {
	return openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: param.NewOpt("I'm out of office right now. Please try again later."),
			},
			Name: param.NewOpt(agent.memberRecord.Name),
		},
	}
}
