package openteam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openteam/entities"
	"go.uber.org/zap"
)

const everyonePrefix string = "Everyone! "

type mention struct {
	channelName    string
	fromMemberName string
	fromRoleID     string
	fromTaskID     string
	toMemberNames  []string
	allMemberNames []string
	message        string
}

func (a mention) addressingToPrefix() (prefix string) {

	allNamesExceptAuthor := make(map[string]struct{})
	for _, memberName := range a.allMemberNames {
		if memberName != a.fromMemberName {
			allNamesExceptAuthor[memberName] = struct{}{}
		}
	}

	if len(allNamesExceptAuthor) == 1 {
		prefix = ""
		return
	}

	if len(a.toMemberNames) == 1 {
		prefix = fmt.Sprintf("%s! ", a.toMemberNames[0])
		return
	}

	isExactMatch := true
	for _, toMemberName := range a.toMemberNames {
		if _, exists := allNamesExceptAuthor[toMemberName]; !exists {
			isExactMatch = false
			break
		}
	}

	if isExactMatch {
		prefix = everyonePrefix
		return
	}

	stringBuilder := strings.Builder{}
	for _, toMemberName := range a.toMemberNames {
		stringBuilder.WriteString(toMemberName)
		stringBuilder.WriteString("! ")
	}

	prefix = stringBuilder.String()
	return

}

func (a mention) persistMessage(
	ctx context.Context,
	qtx *entities.Queries,
	stepRecord entities.Step,
	logger *zap.Logger,
) (
	messageId string,
	err error,
) {

	mentionMessage := fmt.Sprintf("%s%s", a.addressingToPrefix(), a.message)

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
		StepID:        stepRecord.ID,
		ChannelName:   a.channelName,
		RoleID:        a.fromRoleID,
		TaskID:        a.fromTaskID,
	}
	_, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create new message", zap.Error(err))
		return
	}

	messageId = ulid.Make().String()
	return
}

func (a mention) persistMentions(
	ctx context.Context,
	qtx *entities.Queries,
	runRecord entities.Run, // this is a bug
	logger *zap.Logger,
) (
	orchestrationPlan Plan,
	err error,
) {

	for _, roleRecord := range a.toMemberNames {
		if roleRecord == a.fromMemberName {
			continue
		}

		toMemberRecord, er := qtx.GetMember(ctx, roleRecord)
		err = er
		if err != nil {
			logger.Error("failed to get agent member", zap.Error(err))
			return
		}

		createMentionParams := entities.CreateMentionParams{
			ID:               ulid.Make().String(),
			RunID:            runRecord.ID,
			FromMemberTaskID: a.fromTaskID,
			ToMemberName:     toMemberRecord.Name,
			Message:          a.message,
		}

		mentionRecord, er := qtx.CreateMention(ctx, createMentionParams)
		err = er
		if err != nil {
			logger.Error("failed to create mention", zap.Error(err))
			return
		}
		orchestrationPlan.mentionRecords = append(orchestrationPlan.mentionRecords, mentionRecord)
	}

	return

}
