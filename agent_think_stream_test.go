package openteam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"net/http"
	"testing"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openteam/entities"
	"github.com/stretchr/testify/assert"
	"github.com/wiremock/go-wiremock"
)

func Test_Agent_should_call_llm_when_thinking_in_stream_mode(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_when_thinking.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true
		}`

	responseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi! "},"finish_reason":null}]}`
	responseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"content":"I am Jane."},"finish_reason":null}]}`
	responseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	responseChunk4 :=
		`[DONE]`

	responseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\ndata: %s\n\n", responseChunk1, responseChunk2, responseChunk3, responseChunk4)

	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
		InScenario("First Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "text/event-stream").
				WithHeader("Cache-Control", "no-cache").
				WithHeader("Connection", "keep-alive").
				WithBody(responseBodySse),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(requestStub)
	assert.NoError(t, err)

	reply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, 1, len(reply.replyTurnIds))
	actualReplyTurnId := reply.replyTurnIds[0]

	actualReplyMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualReplyTurnId)
	assert.NoError(t, err)
	assert.Equal(t, actualReplyMessage.Visibility, string(VisibilityChannel))
	actualReplyOpenAiMessage := openai.ChatCompletionMessageParamUnion{}
	err = json.Unmarshal(actualReplyMessage.OpenaiMessage, &actualReplyOpenAiMessage)
	actualContent := actualReplyOpenAiMessage.OfUser.Content.OfString.Value
	assert.Equal(t, "Hi! I am Jane.", actualContent)

	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyRequestStub)

}

func Test_Agent_should_call_llm_for_the_followup_conversation_when_thinking_in_stream_mode(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_for_the_followup_conversation_when_thinking.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true
		}`

	firstResponseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi! "},"finish_reason":null}]}`
	firstResponseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"content":"I am Jane."},"finish_reason":null}]}`
	firstResponseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	firstResponseChunk4 :=
		`[DONE]`

	firstResponseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\ndata: %s\n\n", firstResponseChunk1, firstResponseChunk2, firstResponseChunk3, firstResponseChunk4)

	firstRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(firstRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "text/event-stream").
				WithHeader("Cache-Control", "no-cache").
				WithHeader("Connection", "keep-alive").
				WithBody(firstResponseBodySse),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(firstRequestStub)
	assert.NoError(t, err)

	firstReply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(firstReply.replyTurnIds))
	actualFirstReplyTurnId := firstReply.replyTurnIds[0]

	actualFirstReplyMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualFirstReplyTurnId)
	assert.NoError(t, err)
	assert.Equal(t, actualFirstReplyMessage.Visibility, string(VisibilityChannel))
	actualFirstReplyOpenAiMessage := openai.ChatCompletionMessageParamUnion{}
	err = json.Unmarshal(actualFirstReplyMessage.OpenaiMessage, &actualFirstReplyOpenAiMessage)
	actualContent := actualFirstReplyOpenAiMessage.OfUser.Content.OfString.Value
	assert.Equal(t, "Hi! I am Jane.", actualContent)

	secondRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"},
				{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"},
				{"role": "user", "content": "Jane! How are you?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true
		}`

	secondResponseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"I'm fine, thank you!"},"finish_reason":null}]}`
	secondResponseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	secondResponseChunk3 :=
		`[DONE]`

	secondResponseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\n", secondResponseChunk1, secondResponseChunk2, secondResponseChunk3)

	secondRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(secondRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs("first-message-received").
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "text/event-stream").
				WithHeader("Cache-Control", "no-cache").
				WithHeader("Connection", "keep-alive").
				WithBody(secondResponseBodySse),
		).
		WillSetStateTo("second-message-received")

	err = wiremockClient.StubFor(secondRequestStub)
	assert.NoError(t, err)

	secondReply, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondReply)
	assert.Equal(t, 1, len(secondReply.replyTurnIds))
	actualSecondReplyTurnId := secondReply.replyTurnIds[0]

	actualSecondReplyMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualSecondReplyTurnId)
	assert.NoError(t, err)
	assert.Equal(t, actualSecondReplyMessage.Visibility, string(VisibilityChannel))
	actualSecondReplyOpenAiMessage := openai.ChatCompletionMessageParamUnion{}
	err = json.Unmarshal(actualSecondReplyMessage.OpenaiMessage, &actualSecondReplyOpenAiMessage)
	actualContent = actualSecondReplyOpenAiMessage.OfUser.Content.OfString.Value
	assert.Equal(t, "I'm fine, thank you!", actualContent)

	verifyFirstRequestStub, err := wiremockClient.Verify(firstRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyFirstRequestStub)

	verifySecondRequestStub, err := wiremockClient.Verify(secondRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifySecondRequestStub)

}

func Test_Agent_should_persist_the_conversation_history_for_the_first_message_when_thinking_in_stream_mode(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_first_message_when_thinking.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true
		}`

	responseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi! "},"finish_reason":null}]}`
	responseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"content":"I am Jane."},"finish_reason":null}]}`
	responseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	responseChunk4 :=
		`[DONE]`

	responseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\ndata: %s\n\n", responseChunk1, responseChunk2, responseChunk3, responseChunk4)

	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
		InScenario("First Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "text/event-stream").
				WithHeader("Cache-Control", "no-cache").
				WithHeader("Connection", "keep-alive").
				WithBody(responseBodySse),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(requestStub)
	assert.NoError(t, err)

	reply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, 1, len(reply.replyTurnIds))

	allTurns, err := teamDb.Queries.GetTurns(ctx)
	assert.NoError(t, err)

	actualTurn1 := allTurns[0]
	assert.Equal(t, actualTurn1.Kind, string(EventKindAddressing))

	actualTurn2 := allTurns[1]
	assert.Equal(t, actualTurn2.Kind, string(EventKindThinking))
	actualMessage1, err := teamDb.Queries.GetMessageByTurn(ctx, actualTurn2.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage1.Visibility, string(VisibilityChannel))
	actualMessage1Json, err := json.Marshal(actualMessage1.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage1Json := fmt.Sprintf(`{"name":"Jim","content":"Jane! Hello!","role":"user"}`)
	assert.JSONEq(t, expectedMessage1Json, string(actualMessage1Json))

	actualTurn3 := allTurns[2]
	assert.Equal(t, actualTurn3.Kind, string(EventKindPlanning))
	actualLlmChunkRecords, err := teamDb.Queries.GetLlmChunkResponseByTurn(ctx, actualTurn3.ID)
	assert.NoError(t, err)
	assert.Len(t, actualLlmChunkRecords, 3)
	actualLlmChunk1Json, err := json.Marshal(actualLlmChunkRecords[0].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, responseChunk1, string(actualLlmChunk1Json))
	actualLlmChunk2Json, err := json.Marshal(actualLlmChunkRecords[1].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, responseChunk2, string(actualLlmChunk2Json))
	actualLlmChunk3Json, err := json.Marshal(actualLlmChunkRecords[2].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, responseChunk3, string(actualLlmChunk3Json))

	actualTurn4 := allTurns[3]
	assert.Equal(t, actualTurn4.Kind, string(EventKindReplying))
	actualReplying1, err := teamDb.Queries.GetMessageByTurn(ctx, actualTurn4.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying1.Visibility, string(VisibilityChannel))
	actualReplying1Json, err := json.Marshal(actualReplying1.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying1Json := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedReplying1Json, string(actualReplying1Json))

	assert.Len(t, allTurns, 4)

}

func Test_Agent_should_persist_the_conversation_history_for_the_followup_conversation_when_thinking_in_stream_mode(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_followup_conversation_when_thinking.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true
		}`

	firstResponseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"Hi! "},"finish_reason":null}]}`
	firstResponseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"content":"I am Jane."},"finish_reason":null}]}`
	firstResponseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	firstResponseChunk4 :=
		`[DONE]`

	firstResponseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\ndata: %s\n\n", firstResponseChunk1, firstResponseChunk2, firstResponseChunk3, firstResponseChunk4)

	firstRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(firstRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "text/event-stream").
				WithHeader("Cache-Control", "no-cache").
				WithHeader("Connection", "keep-alive").
				WithBody(firstResponseBodySse),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(firstRequestStub)
	assert.NoError(t, err)

	firstReply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(firstReply.replyTurnIds))

	secondRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"},
				{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"},
				{"role": "user", "content": "Jane! How are you?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true
		}`

	secondResponseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"I'm fine, thank you!"},"finish_reason":null}]}`
	secondResponseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	secondResponseChunk3 :=
		`[DONE]`

	secondResponseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\n", secondResponseChunk1, secondResponseChunk2, secondResponseChunk3)

	secondRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(secondRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs("first-message-received").
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "text/event-stream").
				WithHeader("Cache-Control", "no-cache").
				WithHeader("Connection", "keep-alive").
				WithBody(secondResponseBodySse),
		).
		WillSetStateTo("second-message-received")

	err = wiremockClient.StubFor(secondRequestStub)
	assert.NoError(t, err)

	secondReply, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(secondReply.replyTurnIds))

	allTurns, err := teamDb.Queries.GetTurns(ctx)
	assert.NoError(t, err)

	actualTurn1 := allTurns[0]
	assert.Equal(t, actualTurn1.Kind, string(EventKindAddressing))

	actualTurn2 := allTurns[1]
	assert.Equal(t, actualTurn2.Kind, string(EventKindThinking))
	actualMessage1, err := teamDb.Queries.GetMessageByTurn(ctx, actualTurn2.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage1.Visibility, string(VisibilityChannel))
	actualMessage1Json, err := json.Marshal(actualMessage1.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage1Json := fmt.Sprintf(`{"name":"Jim","content":"Jane! Hello!","role":"user"}`)
	assert.JSONEq(t, expectedMessage1Json, string(actualMessage1Json))

	actualTurn3 := allTurns[2]
	assert.Equal(t, actualTurn3.Kind, string(EventKindPlanning))
	actualLlmChunkRecords, err := teamDb.Queries.GetLlmChunkResponseByTurn(ctx, actualTurn3.ID)
	assert.NoError(t, err)
	assert.Len(t, actualLlmChunkRecords, 3)
	actualLlmChunk1Json, err := json.Marshal(actualLlmChunkRecords[0].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseChunk1, string(actualLlmChunk1Json))
	actualLlmChunk2Json, err := json.Marshal(actualLlmChunkRecords[1].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseChunk2, string(actualLlmChunk2Json))
	actualLlmChunk3Json, err := json.Marshal(actualLlmChunkRecords[2].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseChunk3, string(actualLlmChunk3Json))

	actualTurn4 := allTurns[3]
	assert.Equal(t, actualTurn4.Kind, string(EventKindReplying))
	actualReplying1, err := teamDb.Queries.GetMessageByTurn(ctx, actualTurn4.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying1.Visibility, string(VisibilityChannel))
	actualReplying1Json, err := json.Marshal(actualReplying1.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying1Json := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedReplying1Json, string(actualReplying1Json))

	actualTurn5 := allTurns[4]
	assert.Equal(t, actualTurn5.Kind, string(EventKindAddressing))

	actualTurn6 := allTurns[5]
	assert.Equal(t, actualTurn2.Kind, string(EventKindThinking))
	actualMessage2, err := teamDb.Queries.GetMessageByTurn(ctx, actualTurn6.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage2.Visibility, string(VisibilityChannel))
	actualMessage2Json, err := json.Marshal(actualMessage2.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage2Json := fmt.Sprintf(`{"name":"Jim","content":"Jane! How are you?","role":"user"}`)
	assert.JSONEq(t, expectedMessage2Json, string(actualMessage2Json))

	actualTurn7 := allTurns[6]
	assert.Equal(t, actualTurn7.Kind, string(EventKindPlanning))
	actualLlmChunkRecords, err = teamDb.Queries.GetLlmChunkResponseByTurn(ctx, actualTurn7.ID)
	assert.NoError(t, err)
	assert.Len(t, actualLlmChunkRecords, 2)
	actualLlmChunk1Json, err = json.Marshal(actualLlmChunkRecords[0].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseChunk1, string(actualLlmChunk1Json))
	actualLlmChunk2Json, err = json.Marshal(actualLlmChunkRecords[1].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseChunk2, string(actualLlmChunk2Json))

	actualTurn8 := allTurns[7]
	assert.Equal(t, actualTurn4.Kind, string(EventKindReplying))
	actualReplying2, err := teamDb.Queries.GetMessageByTurn(ctx, actualTurn8.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying2.Visibility, string(VisibilityChannel))
	actualReplying2Json, err := json.Marshal(actualReplying2.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying2Json := `{"name":"Jane","content":"I'm fine, thank you!","role":"user"}`
	assert.JSONEq(t, expectedReplying2Json, string(actualReplying2Json))

	assert.Len(t, allTurns, 8)

}

func makeTeamForCallLLMTestInStreamMode(ctx context.Context, teamDb *TeamDb) (err error) {
	q := teamDb.Queries

	jane, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "Jane",
		Kind: "bot",
	})
	if err != nil {
		testLogger.Error("failed to create member", zap.Error(err))
		return err
	}

	john, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "John",
		Kind: "bot",
	})
	if err != nil {
		testLogger.Error("failed to create member", zap.Error(err))
		return err
	}

	jim, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "Jim",
		Kind: "human",
	})
	if err != nil {
		testLogger.Error("failed to create member", zap.Error(err))
		return err
	}

	lobby, err := q.CreateChannel(ctx, entities.CreateChannelParams{
		Name:        "lobby",
		Description: "Lobby channel",
	})
	if err != nil {
		testLogger.Error("failed to create channel", zap.Error(err))
		return err
	}

	warRoom, err := q.CreateChannel(ctx, entities.CreateChannelParams{
		Name:        "war-room",
		Description: "War room channel",
	})
	if err != nil {
		testLogger.Error("failed to create channel", zap.Error(err))
		return err
	}

	janeLobbyRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "jane-at-lobby",
		MemberName:  jane.Name,
		ChannelName: lobby.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	janeWarRoomRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "jane-at-war-room",
		MemberName:  jane.Name,
		ChannelName: warRoom.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	janeLobbyFirstImpression, err := q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jane-lobby-first-impression",
		RoleID:      janeLobbyRole.ID,
		Instruction: "You are a first impression in the lobby.",
		Model:       "gpt-5",
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	janeLobbyDecisionMake, err := q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jane-lobby-decision-make",
		RoleID:      janeLobbyRole.ID,
		PrevID:      sql.NullString{String: janeLobbyFirstImpression.ID, Valid: true},
		Instruction: "You are a decision maker in the lobby.",
		Model:       "gpt-5",
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	_, err = q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jane-war-room-coordinator",
		RoleID:      janeWarRoomRole.ID,
		PrevID:      sql.NullString{String: janeLobbyDecisionMake.ID, Valid: true},
		Instruction: "You are a coordinator in the war room.",
		Model:       "gpt-5",
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	johnWarRoomRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "john-at-war-room",
		MemberName:  john.Name,
		ChannelName: warRoom.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	_, err = q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "john-war-room-expert",
		RoleID:      johnWarRoomRole.ID,
		Instruction: "You are a expert in the war room.",
		Model:       "gpt-5",
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	jimLobbyRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "jim-at-lobby",
		MemberName:  jim.Name,
		ChannelName: lobby.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	_, err = q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jim-the-user",
		RoleID:      jimLobbyRole.ID,
		Instruction: "You are the user.",
		Model:       "gpt-5",
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	return nil
}
