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

func (a *Agent) reply(
	ctx context.Context,
	turnID string,
	logger *zap.Logger,
) (
	reply Reply,
	err error,
) {
	logger = logger.With(zap.String("currentTurnId", turnID))
	logger.Info("getting current turn...")

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

	turnRecord, err := qtx.GetTurn(ctx, turnID)
	if err != nil {
		logger.Error("failed to get turn", zap.Error(err))
		return
	}

	mention, err := reconstituteMention(ctx, qtx, turnRecord, logger)
	if err != nil {
		logger.Error("failed to reconstitute replying context", zap.Error(err))
		return
	}

	thinkingTurnRecord, err := mention.persistMentionMessage(ctx, qtx, turnRecord, logger)
	if err != nil {
		logger.Error("failed to persist mention message", zap.Error(err))
		return
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	reply, err = a.think(ctx, thinkingTurnRecord, mention, logger)
	if err != nil {
		logger.Error("failed to think", zap.Error(err))
		return
	}

	return
}

type Mention struct {
	mentionID      string
	turnID         string
	channelName    string
	fromMemberName string
	fromRoleID     string
	fromTaskID     string
	toMemberName   string
	toRoleID       string
	toTask         entities.Task
	message        string
}

func (a Mention) persistMentionMessage(
	ctx context.Context,
	qtx *entities.Queries,
	turnRecord entities.Turn,
	logger *zap.Logger,
) (
	thinkingTurnRecord entities.Turn,
	err error,
) {

	thinkingTurnRecord, err = createTurn(ctx, qtx, EventKindThinking, logger)
	if err != nil {
		logger.Error("failed to create thinking turn", zap.Error(err))
		return
	}

	err = linkTurns(ctx, qtx, turnRecord.ID, thinkingTurnRecord.ID, logger)
	if err != nil {
		logger.Error("failed to link turns", zap.Error(err))
		return
	}

	mentionMessage := fmt.Sprintf("%s! %s", a.toMemberName, a.message)

	openAiMessage := openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: param.NewOpt(mentionMessage),
			},
			Name: param.NewOpt(a.fromMemberName),
		},
	}

	openAiMessageBytes, err := json.Marshal(openAiMessage)
	if err != nil {
		logger.Error("failed to marshal openai message", zap.Error(err))
		return
	}

	createMessageParams := entities.CreateMessageParams{
		ID:            ulid.Make().String(),
		OpenaiMessage: json.RawMessage(openAiMessageBytes),
		Visibility:    string(VisibilityChannel),
		TurnID:        thinkingTurnRecord.ID,
		ChannelName:   a.channelName,
		RoleID:        a.toRoleID,
		TaskID:        a.toTask.ID,
	}
	_, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create new message", zap.Error(err))
		return
	}

	return
}

func reconstituteMention(
	ctx context.Context,
	qtx *entities.Queries,
	turnRecord entities.Turn,
	logger *zap.Logger,
) (
	mention Mention,
	err error,
) {

	logger = logger.With(zap.String("turnId", turnRecord.ID))

	mentionRecord, err := qtx.GetMentionByTurn(ctx, turnRecord.ID)
	if err != nil {
		logger.Error("failed to get mention", zap.Error(err))
		return
	}

	fromRoleRecord, err := qtx.GetRoleByTask(ctx, mentionRecord.FromMemberTaskID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("roleId", fromRoleRecord.ID))

	channelRecord, err := qtx.GetChannelByRole(ctx, fromRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("channelId", channelRecord.Name))

	fromMemberRecord, err := qtx.GetMemberByTask(ctx, mentionRecord.FromMemberTaskID)
	if err != nil {
		logger.Error("failed to get from member", zap.Error(err))
		return
	}

	toMemberRecord, err := qtx.GetMember(ctx, mentionRecord.ToMemberName)
	if err != nil {
		logger.Error("failed to get to participant", zap.Error(err))
		return
	}

	toRoleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  toMemberRecord.Name,
		ChannelName: channelRecord.Name,
	}

	toRoleRecord, err := qtx.GetRoleByMemberAndChannel(ctx, toRoleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get to membership", zap.Error(err))
		return
	}

	toTaskRecord, err := qtx.GetFirstTask(ctx, toRoleRecord.ID)
	if err != nil {
		logger.Error("failed to get to persona", zap.Error(err))
		return
	}

	mention = Mention{
		mentionID:      mentionRecord.ID,
		turnID:         turnRecord.ID,
		channelName:    channelRecord.Name,
		fromMemberName: fromMemberRecord.Name,
		fromRoleID:     fromRoleRecord.ID,
		fromTaskID:     mentionRecord.FromMemberTaskID,
		toMemberName:   toMemberRecord.Name,
		toRoleID:       toRoleRecord.ID,
		toTask:         toTaskRecord,
		message:        mentionRecord.Message,
	}

	return
}

type Reply struct {
	replyTurnIds []string
	actionIds    []string
}
