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

type mentions struct {
	channelName    string
	fromMemberName string
	fromRoleID     string
	toMemberNames  []string
	allMemberNames []string
	message        string
}

func (mention mentions) addressingToPrefix() (prefix string) {

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

func (mention mentions) createMessageParams(
	ctx context.Context,
	logger *zap.Logger,
) (
	createMessageParams entities.CreateMessageParams,
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

	createMessageParams = entities.CreateMessageParams{
		ID:            ulid.Make().String(),
		OpenaiMessage: json.RawMessage(openAiMessageBytes),
		Visibility:    string(VisibilityChannel),
		ChannelName:   mention.channelName,
		RoleID:        mention.fromRoleID,
	}

	return
}
