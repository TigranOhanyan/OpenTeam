package openteam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

var UnexpectedMessageStructureError = errors.New("unexpected message structure")
var InvalidMentionArgumentsError = errors.New("invalid mention arguments")

type Plan struct {
	mentionStepIds []string
	actingStepIds  []string
}

func (o *Plan) isOnlyControl() bool {
	return len(o.mentionStepIds) != 0 && len(o.actingStepIds) == 0
}

func (o *Plan) isFinalReply() bool {
	return len(o.mentionStepIds) == 0 && len(o.actingStepIds) == 0
}

func (a *Agent) analyze(
	ctx context.Context,
	stepRecord entities.Step,
	mention Mention,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {

	logger = logger.With(zap.String("stepId", stepRecord.ID))
	logger.Info("getting current step...")

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

	llmResponseAsMessage, messageId, err := a.llmResponseToMessage(ctx, stepRecord.ID, mention.toTask, logger)
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

	mentionFactory := MentionFactory{
		fromMention: mention,
		qtx:         qtx,
		stream:      a.ChangeStream,
	}

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
		if strings.EqualFold(function.Function.Name, addressToAgentFunction.Name) {

			mentionStepId, er := mentionFactory.persistMention(ctx, args, function.ID, logger)
			err = er
			if err != nil {
				logger.Error("failed to persist mention", zap.Error(err))
				return
			}

			orchestrationPlan.mentionStepIds = append(orchestrationPlan.mentionStepIds, mentionStepId)
		} else {
			err = UnexpectedMessageStructureError
			return
		}
	}

	if !orchestrationPlan.isOnlyControl() {
		logger.Info("persisting message...")

		replyStepRecord, er := createStep(ctx, qtx, EventKindReplying, logger)
		err = er
		if err != nil {
			logger.Error("failed to create replying step", zap.Error(err))
			return
		}
		err = linkSteps(ctx, qtx, stepRecord.ID, replyStepRecord.ID, logger)
		if err != nil {
			logger.Error("failed to link steps", zap.Error(err))
			return
		}

		setName(&llmResponseAsMessage, mention.toMemberName)

		stepIntoUserMessage(&llmResponseAsMessage)

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
			StepID:        replyStepRecord.ID,
			ChannelName:   mention.channelName,
			RoleID:        mention.toRoleID,
			TaskID:        mention.toTask.ID,
			OpenaiMessage: json.RawMessage(filteredLlmResponseAsMessageBytes),
		}

		_, err = qtx.CreateMessage(ctx, createMessageParams)
		if err != nil {
			logger.Error("failed to create new message", zap.Error(err))
			return
		}

		reply.replyStepIds = append(reply.replyStepIds, replyStepRecord.ID)
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	if orchestrationPlan.isFinalReply() {
		return
	}

	for _, mentionStepId := range orchestrationPlan.mentionStepIds {
		mentionReply, er := a.reply(ctx, mentionStepId, logger) // short circuit if the reply requries tool calls
		err = er
		if err != nil {
			logger.Error("failed to reply to mention", zap.Error(err))
			return
		}
		reply.actionIds = append(reply.actionIds, mentionReply.actionIds...)
	}

	thinkingStepRecord, err := createStep(ctx, qtx, EventKindThinking, logger)

	err = linkSteps(ctx, qtx, stepRecord.ID, thinkingStepRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link steps", zap.Error(err))
		return
	}

	reply, err = a.think(ctx, thinkingStepRecord, mention, logger)
	if err != nil {
		logger.Error("failed to think", zap.Error(err))
		return
	}
	return

}

type MentionFactory struct {
	fromMention Mention
	qtx         *entities.Queries
	stream      chan<- ChangeEvent
}

func (a *MentionFactory) persistMention(
	ctx context.Context,
	arguments map[string]interface{},
	toolCallId string,
	logger *zap.Logger,
) (
	mentionStepId string,
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

	stepRecord, err := createStep(ctx, a.qtx, EventKindMention, logger)

	if err != nil {
		return
	}

	err = linkSteps(ctx, a.qtx, a.fromMention.stepID, stepRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link steps", zap.Error(err))
		return
	}

	createMentionParams := entities.CreateMentionParams{
		ID:               ulid.Make().String(),
		StepID:           stepRecord.ID,
		ToolCallID:       toolCallId,
		FromMemberTaskID: a.fromMention.toTask.ID,
		ToMemberName:     toMemberName,
		Message:          message,
	}

	mentionRecord, err := a.qtx.CreateMention(ctx, createMentionParams)
	if err != nil {
		return
	}

	a.stream <- ChangeEvent{
		Kind:        CdcEventKindMention,
		StepID:      stepRecord.ID,
		ChannelName: a.fromMention.channelName,
		Mention: &entities.Mention{
			ID:               mentionRecord.ID,
			StepID:           stepRecord.ID,
			FromMemberTaskID: mentionRecord.FromMemberTaskID,
			ToMemberName:     mentionRecord.ToMemberName,
			ToolCallID:       mentionRecord.ToolCallID,
			Message:          mentionRecord.Message,
		},
	}

	mentionStepId = stepRecord.ID

	return

}

func (a *Agent) llmResponseToMessage(
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
		llmChunkResponseRecords, er := a.ConversationHistoryDb.Queries.GetLlmChunkResponseByStep(ctx, stepId)
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
		llmResponseRecord, er := a.ConversationHistoryDb.Queries.GetLlmResponseByStep(ctx, stepId)
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

func stepIntoUserMessage(
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
