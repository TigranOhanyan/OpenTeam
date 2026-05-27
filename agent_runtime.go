package OpenTeam

import (
	"github.com/openai/openai-go/v3"
)

type AgentRuntime struct {
	ConversationHistoryDb *TeamDb
	LlmClient             *openai.Client
	ChangeStream          chan<- ChangeEvent
}
