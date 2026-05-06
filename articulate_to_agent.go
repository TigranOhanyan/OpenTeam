package openteam

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func (a *Agent) articulateToAgent(
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
	logger = logger.With(zap.String("previousTurnId", previousTurnRecord.ID))

	dutyOfPreviousTurnRecord, err := a.ConversationHistoryDb.Queries.GetDutyByTurn(ctx, previousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get persona", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("dutyId", dutyOfPreviousTurnRecord.ID))

	roleOfPreviousTurnRecord, err := a.ConversationHistoryDb.Queries.GetRoleByDuty(ctx, dutyOfPreviousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("roleId", roleOfPreviousTurnRecord.ID))

	channelRecord, err := a.ConversationHistoryDb.Queries.GetChannelByRole(ctx, roleOfPreviousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("channelId", channelRecord.Name))

	createTurnParams := entities.CreateTurnParams{
		ID:     ulid.Make().String(),
		Kind:   string(EventKindArticulation),
		Status: string(TurnStatusPending),
	}

	currentTurnRecord, err := a.ConversationHistoryDb.Queries.CreateTurn(ctx, createTurnParams)
	if err != nil {
		logger.Error("failed to create current turn", zap.Error(err))
		return
	}

	articulationRecord, err := a.ConversationHistoryDb.Queries.GetArticulationByTurn(ctx, previousTurnRecord.ID)
	if err != nil {
		logger.Error("failed to get articulation", zap.Error(err))
		return
	}

	fromMemberRecord, err := a.ConversationHistoryDb.Queries.GetMember(ctx, articulationRecord.FromMemberName)
	if err != nil {
		logger.Error("failed to get from member", zap.Error(err))
		return
	}

	toMemberRecord, err := a.ConversationHistoryDb.Queries.GetMember(ctx, articulationRecord.ToMemberName)
	if err != nil {
		logger.Error("failed to get to participant", zap.Error(err))
		return
	}

	toRoleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  toMemberRecord.Name,
		ChannelName: channelRecord.Name,
	}

	toRoleRecord, err := a.ConversationHistoryDb.Queries.GetRoleByMemberAndChannel(ctx, toRoleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get to membership", zap.Error(err))
		return
	}

	toDutyRecord, err := a.ConversationHistoryDb.Queries.GetFirstDuty(ctx, toRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get to persona", zap.Error(err))
		return
	}

	openAiMessage := openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: param.NewOpt(articulationRecord.Message),
			},
			Name: param.NewOpt(fromMemberRecord.Name),
		},
	}

	openAiMessageBytes, err := json.Marshal(openAiMessage)
	if err != nil {
		logger.Error("failed to marshal openai message", zap.Error(err))
		return
	}

	completedAt := time.Now().UTC()

	createMessageParams := entities.CreateMessageParams{
		ID:            ulid.Make().String(),
		OpenaiMessage: json.RawMessage(openAiMessageBytes),
		Visibility:    string(VisibilityChannel),
		TurnID:        currentTurnRecord.ID,
		ChannelName:   channelRecord.Name,
		RoleID:        toRoleRecord.ID,
		DutyID:        toDutyRecord.ID,
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

	currentTurnId = currentTurnRecord.ID
	return

}
