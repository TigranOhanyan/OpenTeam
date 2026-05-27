package OpenTeam

import (
	"slices"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

var mentionMemberFunction = openai.FunctionDefinitionParam{
	Name:        "mention_member",
	Description: param.NewOpt("Call this to mention the agent."),
	Parameters: openai.FunctionParameters{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_name": map[string]string{
				"type": "string",
			},
			"message": map[string]string{
				"type": "string",
			},
		},
		"required": []string{"agent_name", "message"},
	},
	Strict: param.NewOpt(true),
}

var handoffToAgentFunction = openai.FunctionDefinitionParam{
	Name:        "handoff_to_agent",
	Description: param.NewOpt("Call this to handoff to the agent."),
	Parameters: openai.FunctionParameters{
		"type": "object",
		"properties": map[string]interface{}{
			"agent_name": map[string]string{
				"type": "string",
			},
		},
		"required": []string{"agent_name"},
	},
	Strict: param.NewOpt(true),
}

var controlToolCallNames = []string{mentionMemberFunction.Name, handoffToAgentFunction.Name}

func filterControlToolCalls(
	llmResponseAsMessage openai.ChatCompletionMessageParamUnion,
) (filteredLlmResponseAsMessage openai.ChatCompletionMessageParamUnion) {
	filteredLlmResponseAsMessage = llmResponseAsMessage
	if param.IsOmitted(llmResponseAsMessage.OfAssistant) {
		return
	}
	filteredToolCalls := []openai.ChatCompletionMessageToolCallUnionParam{}
	for _, toolCall := range llmResponseAsMessage.OfAssistant.ToolCalls {
		if param.IsOmitted(toolCall.OfFunction) {
			continue
		}
		function := toolCall.OfFunction
		if slices.Contains(controlToolCallNames, function.Function.Name) {
			continue
		}
		filteredToolCalls = append(filteredToolCalls, toolCall)
	}
	llmResponseAsMessage.OfAssistant.ToolCalls = filteredToolCalls
	return
}
