package OpenTeam

import (
	"encoding/json"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

type plan struct {
	hasMessage    bool
	mentionsPlans []mentionPlan
	actionPlans   []actionPlan
}

type mentionPlan struct {
	agentName string
	message   string
	toolCall  openai.ChatCompletionMessageToolCallUnionParam
}

type actionPlan struct {
	toolCall openai.ChatCompletionMessageToolCallUnionParam
}

func (o *plan) isFinalReply() bool {
	return len(o.mentionsPlans) == 0 && len(o.actionPlans) == 0 && o.hasMessage
}

func (plan *plan) parse(
	message openai.ChatCompletionMessageParamUnion,
) (
	err error,
) {
	assistantMessage := message.OfAssistant
	if param.IsOmitted(assistantMessage) {
		return
	}

	for _, toolCall := range assistantMessage.ToolCalls {
		if param.IsOmitted(toolCall.OfFunction) {
			continue
		}
		function := toolCall.OfFunction
		var args map[string]interface{}
		err = json.Unmarshal([]byte(function.Function.Arguments), &args)
		if err != nil {
			return
		}
		if strings.EqualFold(function.Function.Name, mentionMemberFunction.Name) {

			agentNameCandidate, ok := args["agent_name"]
			if !ok {
				err = InvalidMentionArgumentsError
				return
			}
			agentName, ok := agentNameCandidate.(string)
			if !ok {
				err = InvalidMentionArgumentsError
				return
			}

			messageCandidate, ok := args["message"]
			if !ok {
				err = InvalidMentionArgumentsError
				return
			}
			message, ok := messageCandidate.(string)
			if !ok {
				err = InvalidMentionArgumentsError
				return
			}

			mentionPlan := mentionPlan{
				agentName: agentName,
				message:   message,
				toolCall:  toolCall,
			}

			plan.mentionsPlans = append(plan.mentionsPlans, mentionPlan)
		} else {
			actionPlan := actionPlan{
				toolCall: toolCall,
			}
			plan.actionPlans = append(plan.actionPlans, actionPlan)
		}
	}

	if !param.IsOmitted(assistantMessage.Content.OfString) {
		plan.hasMessage = true
	}

	if len(assistantMessage.Content.OfArrayOfContentParts) > 0 {
		plan.hasMessage = true
	}

	return
}

func (plan *plan) filterControlToolCalls(
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
) (filteredLlmResponseAsMessage openai.ChatCompletionMessageParamUnion) {
	filteredLlmResponseAsMessage = llmResponseAsMessage
	if param.IsOmitted(llmResponseAsMessage.OfAssistant) {
		return
	}
	filteredToolCalls := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(plan.actionPlans))
	for _, actionPlan := range plan.actionPlans {
		filteredToolCalls = append(filteredToolCalls, actionPlan.toolCall)
	}
	llmResponseAsMessage.OfAssistant.ToolCalls = filteredToolCalls
	return
}
