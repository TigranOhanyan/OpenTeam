package openteam

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

type CdcEventKind string

const (
	CdcEventKindAction       CdcEventKind = "action"
	CdcEventKindChunk        CdcEventKind = "chunk"
	CdcEventKindArticulation CdcEventKind = "articulation"
)

type ChangeEvent struct {
	Kind         CdcEventKind
	TurnID       string
	ChannelName  string
	Action       *entities.Action
	Chunk        *entities.LlmChunkResponse
	Articulation *entities.Articulation
}

type Agent struct {
	ConversationHistoryDb *TeamDb
	LlmClient             *openai.Client
	ChangeStream          chan<- ChangeEvent
}

func (a *Agent) Acknowledge(
	ctx context.Context,
	username string,
	channelName string,
	userMessage string,
	logger *zap.Logger,
) (
	messageId string,
	err error,
) {
	defer func() {
		if a.ChangeStream != nil {
			close(a.ChangeStream)
		}
	}()

	userMemberRecord, err := a.ConversationHistoryDb.Queries.GetMember(ctx, username)
	if err != nil {
		logger.Error("failed to get user participant", zap.Error(err))
		return
	}

	channelRecord, err := a.ConversationHistoryDb.Queries.GetChannel(ctx, channelName)
	if err != nil {
		logger.Error("failed to get user room", zap.Error(err))
		return
	}

	roleRecords, err := a.ConversationHistoryDb.Queries.GetRoleByChannel(ctx, channelRecord.Name)
	if err != nil {
		logger.Error("failed to get user membership", zap.Error(err))
		return
	}

	var userRoleRecord *entities.Role
	var agentRoleRecord *entities.Role
	for _, roleRecord := range roleRecords {
		if roleRecord.MemberName == userMemberRecord.Name {
			userRoleRecord = &roleRecord
		} else {
			agentRoleRecord = &roleRecord
		}
	}

	if userRoleRecord == nil {
		err = errors.New("failed to get user role")
		logger.Error("failed to get user role", zap.Error(err))
		return
	}

	if agentRoleRecord == nil {
		err = errors.New("failed to get agent role")
		logger.Error("failed to get agent role", zap.Error(err))
		return
	}

	userDutyRecord, err := a.ConversationHistoryDb.Queries.GetFirstDuty(ctx, userRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get user persona", zap.Error(err))
		return
	}

	agentMemberRecord, err := a.ConversationHistoryDb.Queries.GetMember(ctx, agentRoleRecord.MemberName)
	if err != nil {
		logger.Error("failed to get agent member", zap.Error(err))
		return
	}

	agentDutyRecord, err := a.ConversationHistoryDb.Queries.GetFirstDuty(ctx, agentRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get agent duty", zap.Error(err))
		return
	}

	createTurnParams := entities.CreateTurnParams{
		ID:     ulid.Make().String(),
		Kind:   string(EventKindThought),
		Status: string(TurnStatusPending),
	}

	userTurnRecord, err := a.ConversationHistoryDb.Queries.CreateTurn(ctx, createTurnParams)
	if err != nil {
		logger.Error("failed to create turn", zap.Error(err))
		return
	}

	agentMemeber, err := a.ConversationHistoryDb.Queries.GetMemberByDuty(ctx, agentDutyRecord.ID)
	if err != nil {
		logger.Error("failed to get agent name", zap.Error(err))
		return
	}

	unifiedId := ulid.Make().String()
	toolCallId := unifiedId // this may not work if we get more than one tool call. maybe we have to append the message id to the tool name.
	arguments, err := json.Marshal(map[string]interface{}{
		"agent_name": agentMemeber.Name,
		"message":    userMessage,
	})
	if err != nil {
		logger.Error("failed to marshal arguments", zap.Error(err))
		return
	}
	toolCall := openai.ChatCompletionMessageToolCallUnionParam{
		OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
			ID: toolCallId,
			Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
				Name:      ArticulateToAgentFunction.Name,
				Arguments: string(arguments),
			},
		},
	}

	openAiMessage := openai.ChatCompletionMessageParamUnion{
		OfAssistant: &openai.ChatCompletionAssistantMessageParam{
			ToolCalls: []openai.ChatCompletionMessageToolCallUnionParam{toolCall},
			Name:      param.NewOpt(username),
		},
	}

	openAiMessageBytes, err := json.Marshal(openAiMessage)
	if err != nil {
		logger.Error("failed to marshal openai message", zap.Error(err))
		return
	}

	now := time.Now().UTC()

	messageId = unifiedId
	createMessageParams := entities.CreateMessageParams{
		ID:            messageId,
		OpenaiMessage: openAiMessageBytes,
		Visibility:    string(VisibilityHidden),
		DutyID:        userDutyRecord.ID,
		RoleID:        userRoleRecord.ID,
		ChannelName:   channelRecord.Name,
		TurnID:        userTurnRecord.ID,
	}
	_, err = a.ConversationHistoryDb.Queries.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create message", zap.Error(err))
		return
	}

	articulationId := ulid.Make().String()

	createArticulationParams := entities.CreateArticulationParams{
		ID:             articulationId,
		TurnID:         userTurnRecord.ID,
		ToolCallID:     toolCallId,
		FromMemberName: username,
		ToMemberName:   agentMemberRecord.Name,
		Message:        userMessage,
	}
	err = a.insertArticulation(ctx, createArticulationParams)
	if err != nil {
		logger.Error("failed to create articulation", zap.Error(err))
		return
	}

	updateTurnStatusParams := entities.UpdateTurnStatusParams{
		ID:          userTurnRecord.ID,
		Status:      string(TurnStatusCompleted),
		CompletedAt: sql.NullTime{Time: now, Valid: true},
	}
	_, err = a.ConversationHistoryDb.Queries.UpdateTurnStatus(ctx, updateTurnStatusParams)
	if err != nil {
		logger.Error("failed to update human turn", zap.Error(err))
		return
	}

	return
}

func (a *Agent) Reply(
	ctx context.Context,
	messageId string,
	logger *zap.Logger,
) (
	replyMessageId string,
	reply string,
	replyerName string,
	err error,
) {
	defer func() {
		if a.ChangeStream != nil {
			close(a.ChangeStream)
		}
	}()

	messageRecord, err := a.ConversationHistoryDb.Queries.GetMessage(ctx, messageId)
	if err != nil {
		logger.Error("failed to get reply message", zap.Error(err))
		return
	}

	turnRecord, err := a.ConversationHistoryDb.Queries.GetTurn(ctx, messageRecord.TurnID)
	if err != nil {
		logger.Error("failed to get turn", zap.Error(err))
		return
	}

	lastTurnId, err := a.callAgent(ctx, turnRecord.ID, EventKindArticulation, logger)
	if err != nil {
		logger.Error("failed to call agent", zap.Error(err))
		return
	}

	lastTurnRecord, err := a.ConversationHistoryDb.Queries.GetTurn(ctx, lastTurnId)
	if err != nil {
		logger.Error("failed to get last turn", zap.Error(err))
		return
	}

	lastMessageRecord, err := a.ConversationHistoryDb.Queries.GetMessageByTurn(ctx, lastTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get last message", zap.Error(err))
		return
	}

	var lastOpenaiMessage openai.ChatCompletionMessageParamUnion
	if err = json.Unmarshal(lastMessageRecord.OpenaiMessage, &lastOpenaiMessage); err != nil {
		logger.Error("failed to unmarshal openai message", zap.Error(err))
		return
	}

	// todo: handle array of content parts
	replyMessageId = lastMessageRecord.ID
	reply = lastOpenaiMessage.OfUser.Content.OfString.Value

	replyerDutyRecord, err := a.ConversationHistoryDb.Queries.GetDutyByTurn(ctx, lastMessageRecord.TurnID)
	if err != nil {
		logger.Error("failed to get replyer duty", zap.Error(err))
		return
	}
	replyerMemberRecord, err := a.ConversationHistoryDb.Queries.GetMemberByDuty(ctx, replyerDutyRecord.ID)
	if err != nil {
		logger.Error("failed to get replyer name", zap.Error(err))
		return
	}
	replyerName = replyerMemberRecord.Name

	return
}
