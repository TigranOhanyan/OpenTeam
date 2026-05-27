package OpenTeam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"go.uber.org/zap"
)

var UnexpectedMessageStructureError = errors.New("unexpected message structure")
var InvalidMentionArgumentsError = errors.New("invalid mention arguments")

func (agent *agent) act(
	ctx context.Context,
	reasoningStepRecord entities.Step,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {

	logger = logger.With(zap.String("stepId", reasoningStepRecord.ID))
	logger.Info("getting current step...")

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

	actingStepRecord, err := createStep(ctx, qtx, EventKindActing, &agent.runRecord.ID, logger)
	if err != nil {
		logger.Error("failed to create acting step", zap.Error(err))
		return
	}

	err = linkSteps(ctx, qtx, reasoningStepRecord.ID, actingStepRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link steps", zap.Error(err))
		return
	}

	llmResponseAsMessage, messageId, err := agent.runtime.llmResponseToMessage(ctx, reasoningStepRecord.ID, agent.task, logger)
	if err != nil {
		logger.Error("failed to get llm response as message", zap.Error(err))
		return
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

			mentionOrchestrationPlan, er := agent.persistMention(ctx, qtx, args, function.ID, actingStepRecord, logger)
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
			StepID:        actingStepRecord.ID,
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

		reply.ReplyStepIds = append(reply.ReplyStepIds, actingStepRecord.ID)
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	if orchestrationPlan.isFinalReply() {
		return
	}

	for _, mentionRecord := range orchestrationPlan.mentionRecords {
		nextAgent, er := agent.runtime.createAgent(ctx, mentionRecord, logger)
		err = er
		if err != nil {
			logger.Error("failed to create agent", zap.Error(err))
			return
		}

		reply, err = nextAgent.reAct(ctx, logger)
		if err != nil {
			logger.Error("failed to reAct mention", zap.Error(err))
			return
		}
	}

	reply, err = agent.reason(ctx, actingStepRecord, logger)
	if err != nil {
		logger.Error("failed to think", zap.Error(err))
		return
	}
	return

}

func (agent *agent) persistMention(
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

	mention := mention{
		channelName:    agent.channel.Name,
		fromMemberName: agent.member.Name,
		fromRoleID:     agent.role.ID,
		fromTaskID:     agent.task.ID,
		toMemberNames:  []string{toMemberName},
		allMemberNames: []string{toMemberName},
		message:        message,
	}

	_, err = mention.persistMessage(ctx, qtx, sourceStep, logger)
	if err != nil {
		logger.Error("failed to persist mention message", zap.Error(err))
		return
	}

	orchestrationPlan, err = agent.runtime.persistMentions(ctx, qtx, mention, sourceStep, logger)
	if err != nil {
		logger.Error("failed to persist mention", zap.Error(err))
		return
	}

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
