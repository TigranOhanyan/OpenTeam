package OpenTeam

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"go.uber.org/zap"
)

type reactExecutionResolver struct {
	team *Team
}

func (r *reactExecutionResolver) Resolve(
	ctx context.Context,
	reactExecution entities.Execution,
	logger *zap.Logger,
) (executionReport ExecutionReport, err error) {
	executionReport.ExecutionID = reactExecution.ID

	logger = logger.With(zap.String("reactExecutionId", reactExecution.ID))
	logger.Info("auto resolving react execution...")

	trx, err := r.team.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
			return
		}
		err = trx.Commit()
		if err != nil {
			logger.Error("failed to commit transaction", zap.Error(err))
			return
		}
		return
	}()

	qtx := r.team.ConversationHistoryDb.Queries.WithTx(trx)

	reactLoop, err := qtx.GetReactLoop(ctx, reactExecution.ID)
	if err != nil {
		logger.Error("failed to get react loop", zap.Error(err))
		return
	}

	allChildren, err := qtx.GetChildExecutions(ctx, reactExecution.ID)
	if err != nil {
		logger.Error("failed to get latest child execution", zap.Error(err))
		return
	}
	if len(allChildren) == 0 {
		err = handleReasonExecution(ctx, qtx, reactExecution, reactLoop, logger)
		if err != nil {
			logger.Error("failed to handle reason execution", zap.Error(err))
			return
		}
		executionReport.Status = ExecutionStatusExpanded
		return
	}

	allChildrenClosed := true
	for _, child := range allChildren {
		if child.Status != "closed" {
			allChildrenClosed = false
			break
		}
	}
	if !allChildrenClosed {
		executionReport.Status = ExecutionStatusSkipped
		return
	}

	latestChildExecutionRecord := allChildren[0]

	if latestChildExecutionRecord.Kind == "reason" {
		err = handleActExecution(ctx, qtx, reactExecution, latestChildExecutionRecord, reactLoop, logger)
		if err != nil {
			logger.Error("failed to handle act execution", zap.Error(err))
			return
		}
		executionReport.Status = ExecutionStatusExpanded
		return
	}

	return
}

func handleActExecution(
	ctx context.Context,
	qtx *entities.Queries,
	reactExecution entities.Execution,
	reasonExecution entities.Execution,
	reactLoop entities.ReactLoop,
	logger *zap.Logger,
) (
	err error,
) {

	llmResponseRecord, err := qtx.GetLlmResponse(ctx, reasonExecution.ID)
	if err != nil {
		logger.Error("failed to get llm response", zap.Error(err))
		return
	}

	llmResponseAsMessage, err := llmResponseToMessageParam(ctx, qtx, llmResponseRecord, logger)
	if err != nil {
		logger.Error("failed to convert llm response to param", zap.Error(err))
		return
	}

	taskRecord, err := qtx.GetTask(ctx, reactLoop.TaskID)
	if err != nil {
		logger.Error("failed to get task", zap.Error(err))
		return
	}

	roleRecord, err := qtx.GetRole(ctx, taskRecord.RoleID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}

	setName(&llmResponseAsMessage, roleRecord.MemberName)

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

	if orchestrationPlan.hasActions() {
		rawActExecution := rawExecution{
			kind:     "act",
			parentId: &reasonExecution.ID,
		}

		actExecution, er := rawActExecution.persist(ctx, qtx, logger)
		err = er
		if err != nil {
			logger.Error("failed to persist act execution", zap.Error(err))
			return
		}

		if len(orchestrationPlan.mentionsPlans) > 0 {
			logger.Info("persisting mentions...")

			allRoleRecords, er := qtx.GetRoleByChannel(ctx, roleRecord.ChannelName)
			err = er
			if err != nil {
				logger.Error("failed to get user membership", zap.Error(err))
				return
			}

			allMembersName := make([]string, len(allRoleRecords))
			for i, roleRecord := range allRoleRecords {
				allMembersName[i] = roleRecord.MemberName
			}

			for _, mentionPlan := range orchestrationPlan.mentionsPlans {
				mentions := mentionPlan.rawMentions(roleRecord.ChannelName, roleRecord.MemberName, roleRecord.ID, allMembersName, actExecution.ID)
				_, err = mentions.persist(ctx, qtx, logger)
				if err != nil {
					logger.Error("failed to persist mentions", zap.Error(err))
					return
				}
			}

		}

		for _, actionPlan := range orchestrationPlan.actionPlans {

			actionExecution := actionPlan.rawToolCallExecution(actExecution.ID, taskRecord.ID, llmResponseRecord.ExecutionID)

			_, err = actionExecution.persist(ctx, qtx, logger)
			if err != nil {
				logger.Error("failed to persist action execution", zap.Error(err))
				return
			}

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
			ID:            ulid.Make().String(),
			Visibility:    string(VisibilityChannel),
			ExecutionID:   reasonExecution.ID,
			ChannelName:   roleRecord.ChannelName,
			RoleID:        roleRecord.ID,
			TaskID:        taskRecord.ID,
			OpenaiMessage: json.RawMessage(userLlmResponseAsMessageBytes),
		}

		_, err = qtx.CreateMessage(ctx, createMessageParams)
		if err != nil {
			logger.Error("failed to create new message", zap.Error(err))
			return
		}

	}

	reactLoopStatusParams := entities.UpdateReactLoopStatusParams{
		Status:      "act",
		ExecutionID: reactLoop.ExecutionID,
	}
	_, err = qtx.UpdateReactLoopStatus(ctx, reactLoopStatusParams)
	if err != nil {
		logger.Error("failed to update react loop status", zap.Error(err))
		return
	}

	return

}

func handleReasonExecution(
	ctx context.Context,
	qtx *entities.Queries,
	reactExecution entities.Execution,
	reactLoop entities.ReactLoop,
	logger *zap.Logger,
) (
	err error,
) {

	roleRecord, err := qtx.GetRole(ctx, reactLoop.TaskID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("roleId", roleRecord.ID))

	channelRecord, err := qtx.GetChannel(ctx, roleRecord.ChannelName)
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("channelId", channelRecord.Name))

	memberRecord, err := qtx.GetMember(ctx, roleRecord.MemberName)
	if err != nil {
		logger.Error("failed to get to participant", zap.Error(err))
		return
	}

	rawReActExecution := rawExecution{
		kind:     "reAct",
		parentId: &reactExecution.ID,
	}

	reactExecutionRecord, err := rawReActExecution.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist execution", zap.Error(err))
		return
	}

	logger.Info("getting messages...")

	getContextMessagesParams := entities.GetContextMessagesParams{
		TaskID:      reactLoop.TaskID,
		RoleID:      roleRecord.ID,
		ChannelName: channelRecord.Name,
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
		openAiMessage = turnIntoAssistantMessage(openAiMessage, memberRecord.Name)
		openAiMessages[i] = openAiMessage
	}

	toolRecords, err := qtx.GetToolsByTask(ctx, reactLoop.TaskID)
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
		Model:    "gpt-4o-mini", //TODO try to get rid of the model. maybe we can skip it and json encoder won't complain
		Messages: openAiMessages,
		Tools:    openAiTools,
	}

	rawLLMRequest := rawLLMRequest{
		parentExecutionID: reactExecutionRecord.ID,
		taskID:            reactLoop.TaskID,
		chatParams:        &chatParams,
	}

	_, err = rawLLMRequest.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist llm request", zap.Error(err))
		return
	}

	reactLoopStatusParams := entities.UpdateReactLoopStatusParams{
		Status:      "reason",
		ExecutionID: reactLoop.ExecutionID,
	}
	_, err = qtx.UpdateReactLoopStatus(ctx, reactLoopStatusParams)
	if err != nil {
		logger.Error("failed to update react loop status", zap.Error(err))
		return
	}

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

func llmResponseToMessageParam(
	ctx context.Context,
	qtx *entities.Queries,
	llmResponseRecord entities.LlmResponse,
	logger *zap.Logger,
) (
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
	err error,
) {
	switch llmResponseRecord.Kind {
	case "bulk":
		llmBulkResponseRecord, er := qtx.GetLlmBulkResponse(ctx, llmResponseRecord.ExecutionID)
		err = er
		if err != nil {
			logger.Error("failed to get llm bulk response", zap.Error(err))
			return
		}
		var openAillmResponse openai.ChatCompletion
		err = json.Unmarshal(llmBulkResponseRecord.OpenaiResponse, &openAillmResponse)
		if err != nil {
			logger.Error("failed to unmarshal llm response", zap.Error(err))
			return
		}
		choices := openAillmResponse.Choices
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
	case "chunk":
		llmChunkResponseRecords, er := qtx.GetLlmChunkResponse(ctx, llmResponseRecord.ExecutionID)
		err = er
		if err != nil {
			logger.Error("failed to get llm chunk response", zap.Error(err))
			return
		}

		amountOfChunks := len(llmChunkResponseRecords)
		if amountOfChunks == 0 {
			logger.Error("no chunks in LLM response")
			err = fmt.Errorf("no chunks in LLM response")
			return
		}

		acc := openai.ChatCompletionAccumulator{}
		for _, llmChunkResponseRecord := range llmChunkResponseRecords {
			var openAiChunkResponse openai.ChatCompletionChunk
			err = json.Unmarshal(llmChunkResponseRecord.OpenaiChunkResponse, &openAiChunkResponse)
			if err != nil {
				logger.Error("failed to unmarshal openai chunk response", zap.Error(err))
				return
			}
			acc.AddChunk(openAiChunkResponse)
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
