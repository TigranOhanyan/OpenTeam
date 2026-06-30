package OpenTeam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func dumpExecutions(ctx context.Context, teamDb *TeamDb, label string) {
	execs, _ := teamDb.Queries.GetAllExecutions(ctx)
	fmt.Printf("===== EXECUTIONS @ %s =====\n", label)
	for _, e := range execs {
		fmt.Printf("  id=%s kind=%-8s status=%s\n", e.ID, e.Kind, e.Status)
	}
}

func Test_ZZ_Explore_pipeline(t *testing.T) {
	var err error
	ctx := context.TODO()

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "zz_explore.db", testLogger)
	assert.NoError(t, err)
	defer teamDb.Close()

	err = makeTeamForExplore(ctx, teamDb)
	assert.NoError(t, err)

	teamDb.DB.SetMaxOpenConns(1)

	team := Team{ConversationHistoryDb: teamDb}
	resolver := &generalResolver{team: &team}
	engine := Engine{ConversationHistoryDb: teamDb}

	_, _, err = team.Ask(ctx, "Jim", "lobby", "Hello! What is weather in Yerevan?", testLogger)
	assert.NoError(t, err)
	dumpExecutions(ctx, teamDb, "after Ask")

	toolCallResponse := `{
		"id": "chatcmpl-123","object": "chat.completion","created": 1677652288,"model": "gpt-5",
		"choices": [{"index":0,"message":{"role":"assistant","content":"","tool_calls":[
			{"id":"call_QEtzMbHEUaMIWL3ezXuxRRE5","type":"function","function":{"name":"get_current_weather","arguments":"{\"location\":\"Yerevan, Armenia\",\"unit\":\"celsius\"}"}}
		]},"finish_reason":"tool_calls"}]}`

	for i := 0; i < 12; i++ {
		reports, er := engine.Advance(ctx, testLogger, resolver)
		assert.NoError(t, er)
		dumpExecutions(ctx, teamDb, fmt.Sprintf("after Advance %d (reports=%d)", i, len(reports.Reports)))

		// resolve any open reason execution by injecting an llm response
		execs, _ := teamDb.Queries.GetAllExecutions(ctx)
		resolvedSomething := false
		for _, e := range execs {
			if e.Kind == "reason" && e.Status == "open" {
				var resp openai.ChatCompletion
				_ = json.Unmarshal([]byte(toolCallResponse), &resp)
				er2 := team.ResolveLLMCallExecutionBulk(ctx, e.ID, &resp, testLogger)
				fmt.Printf("  -> resolved reason %s err=%v\n", e.ID, er2)
				resolvedSomething = true
			}
		}
		if resolvedSomething {
			dumpExecutions(ctx, teamDb, fmt.Sprintf("after resolve reason (iter %d)", i))
		}
	}

	// final dump of specialized rows
	actions, _ := teamDb.Queries.GetAllExecutions(ctx)
	fmt.Printf("final execution count=%d\n", len(actions))
}

func makeTeamForExplore(ctx context.Context, teamDb *TeamDb) (err error) {
	q := teamDb.Queries

	jane, err := q.CreateMember(ctx, entities.CreateMemberParams{Name: "Jane", Kind: "bot"})
	if err != nil {
		return err
	}
	jim, err := q.CreateMember(ctx, entities.CreateMemberParams{Name: "Jim", Kind: "human"})
	if err != nil {
		return err
	}
	lobby, err := q.CreateChannel(ctx, entities.CreateChannelParams{Name: "lobby", Description: "Lobby channel"})
	if err != nil {
		return err
	}
	janeLobbyRole, err := q.CreateRole(ctx, entities.CreateRoleParams{ID: "jane-at-lobby", MemberName: jane.Name, ChannelName: lobby.Name})
	if err != nil {
		return err
	}
	janeFirst, err := q.CreateTask(ctx, entities.CreateTaskParams{ID: "jane-lobby-first-impression", RoleID: janeLobbyRole.ID, Instruction: "first"})
	if err != nil {
		return err
	}
	weatherTool := openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        "get_current_weather",
				Description: param.NewOpt("Get the current weather in a given location"),
				Parameters: openai.FunctionParameters{
					"type":       "object",
					"properties": map[string]interface{}{"location": map[string]string{"type": "string"}},
					"required":   []string{"location"},
				},
			},
			Type: "function",
		},
	}
	weatherToolJson, _ := json.Marshal(weatherTool)
	_, err = q.CreateTool(ctx, entities.CreateToolParams{ID: "jane-tool", TaskID: janeFirst.ID, Tool: weatherToolJson})
	if err != nil {
		return err
	}
	jimLobbyRole, err := q.CreateRole(ctx, entities.CreateRoleParams{ID: "jim-at-lobby", MemberName: jim.Name, ChannelName: lobby.Name})
	if err != nil {
		return err
	}
	_, err = q.CreateTask(ctx, entities.CreateTaskParams{ID: "jim-the-user", RoleID: jimLobbyRole.ID, PrevID: sql.NullString{}, Instruction: "user"})
	if err != nil {
		return err
	}
	_ = zap.String
	return nil
}
