package OpenTeam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/oklog/ulid/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"go.uber.org/zap"
)

const everyonePrefix string = "Everyone! "

type rawMentions struct {
	channelName    string
	fromMemberName string
	fromRoleID     string
	toMemberNames  []string
	allMemberNames []string
	message        string
	executionID    string
}

func (mentions rawMentions) addressingToPrefix() (prefix string) {

	allNamesExceptAuthor := make(map[string]struct{})
	for _, memberName := range mentions.allMemberNames {
		if memberName != mentions.fromMemberName {
			allNamesExceptAuthor[memberName] = struct{}{}
		}
	}

	if len(allNamesExceptAuthor) == 1 {
		prefix = ""
		return
	}

	if len(mentions.toMemberNames) == 1 {
		prefix = fmt.Sprintf("%s! ", mentions.toMemberNames[0])
		return
	}

	isExactMatch := true
	for _, toMemberName := range mentions.toMemberNames {
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
	for _, toMemberName := range mentions.toMemberNames {
		stringBuilder.WriteString(toMemberName)
		stringBuilder.WriteString("! ")
	}

	prefix = stringBuilder.String()
	return

}

func (mentions rawMentions) persist(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	messageRecord entities.Message,
	cdcEvents []CdcEvent,
	err error,
) {

	messageRecord, err = mentions.persistMessage(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist message", zap.Error(err))
		return
	}

	for _, memberName := range mentions.toMemberNames {
		if memberName == mentions.fromMemberName {
			continue
		}

		rawMention := rawMention{
			parentExecutionID: messageRecord.ID,
			messageID:         messageRecord.ID,
			fromRoleID:        mentions.fromRoleID,
			toMemberName:      memberName,
			message:           mentions.message,
		}

		mentionRecord, er := rawMention.persist(ctx, qtx, logger)
		err = er
		if err != nil {
			logger.Error("failed to persist mention", zap.Error(err))
			return
		}

		event := CdcEvent{
			Kind:        CdcEventKindMention,
			ChannelName: mentions.channelName,
			MemberName:  memberName,
			Mention:     &mentionRecord,
		}

		cdcEvents = append(cdcEvents, event)

	}

	return

}

func (mention rawMentions) persistMessage(
	ctx context.Context,
	qtx *entities.Queries,
	logger *zap.Logger,
) (
	messageRecord entities.Message,
	err error,
) {

	mentionMessage := fmt.Sprintf("%s%s", mention.addressingToPrefix(), mention.message)

	openAiMessage := openai.ChatCompletionMessageParamUnion{
		OfUser: &openai.ChatCompletionUserMessageParam{
			Content: openai.ChatCompletionUserMessageParamContentUnion{
				OfString: param.NewOpt(mentionMessage),
			},
			Name: param.NewOpt(mention.fromMemberName),
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
		ChannelName:   mention.channelName,
		RoleID:        mention.fromRoleID,
		ExecutionID:   mention.executionID,
	}

	messageRecord, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create message", zap.Error(err))
		return
	}

	return
}
