package openteam

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

var ArticulateToAgentFunction = openai.FunctionDefinitionParam{
	Name:        "articulate_to_agent",
	Description: param.NewOpt("Call this to articulate to the agent."),
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
