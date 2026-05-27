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

type mention struct {
	channelName    string
	fromMemberName string
	fromRoleID     string
	fromTaskID     string
	toMemberNames  []string
	allMemberNames []string
	message        string
}

func (mention mention) addressingToPrefix() (prefix string) {

	allNamesExceptAuthor := make(map[string]struct{})
	for _, memberName := range mention.allMemberNames {
		if memberName != mention.fromMemberName {
			allNamesExceptAuthor[memberName] = struct{}{}
		}
	}

	if len(allNamesExceptAuthor) == 1 {
		prefix = ""
		return
	}

	if len(mention.toMemberNames) == 1 {
		prefix = fmt.Sprintf("%s! ", mention.toMemberNames[0])
		return
	}

	isExactMatch := true
	for _, toMemberName := range mention.toMemberNames {
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
	for _, toMemberName := range mention.toMemberNames {
		stringBuilder.WriteString(toMemberName)
		stringBuilder.WriteString("! ")
	}

	prefix = stringBuilder.String()
	return

}

func (mention mention) persistMessage(
	ctx context.Context,
	qtx *entities.Queries,
	stepRecord entities.Step,
	logger *zap.Logger,
) (
	messageId string,
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
		StepID:        stepRecord.ID,
		ChannelName:   mention.channelName,
		RoleID:        mention.fromRoleID,
		TaskID:        mention.fromTaskID,
	}
	_, err = qtx.CreateMessage(ctx, createMessageParams)
	if err != nil {
		logger.Error("failed to create new message", zap.Error(err))
		return
	}

	messageId = ulid.Make().String()
	return
}

func (runtime *AgentRuntime) persistMentions(
	ctx context.Context,
	qtx *entities.Queries,
	mention mention,
	sourceStep entities.Step,
	logger *zap.Logger,
) (
	orchestrationPlan Plan,
	err error,
) {

	for _, roleRecord := range mention.toMemberNames {
		if roleRecord == mention.fromMemberName {
			continue
		}

		toMemberRecord, er := qtx.GetMember(ctx, roleRecord)
		err = er
		if err != nil {
			logger.Error("failed to get agent member", zap.Error(err))
			return
		}

		nextRunRecord, er := createRun(ctx, qtx, sourceStep.ID, logger)
		err = er
		if err != nil {
			logger.Error("failed to create run", zap.Error(err))
			return
		}

		createMentionParams := entities.CreateMentionParams{
			ID:               ulid.Make().String(),
			RunID:            nextRunRecord.ID,
			FromMemberTaskID: mention.fromTaskID,
			ToMemberName:     toMemberRecord.Name,
			Message:          mention.message,
		}

		mentionRecord, er := qtx.CreateMention(ctx, createMentionParams)
		err = er
		if err != nil {
			logger.Error("failed to create mention", zap.Error(err))
			return
		}

		if runtime.ChangeStream != nil {

			runtime.ChangeStream <- ChangeEvent{
				Kind:        CdcEventKindMention,
				RunID:       nextRunRecord.ID,
				StepID:      sourceStep.ID,
				ChannelName: mention.channelName,
				MemberName:  toMemberRecord.Name,
				TaskID:      mention.fromTaskID,
				Mention:     &mentionRecord,
			}
		}
		orchestrationPlan.mentionRecords = append(orchestrationPlan.mentionRecords, mentionRecord)
	}

	return

}
