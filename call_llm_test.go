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

	"github.com/openteam/entities"
	"github.com/stretchr/testify/assert"
	"github.com/wiremock/go-wiremock"
)

func Test_Agent_should_call_llm(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "aganet_should_call_llm.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTest(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello Jane!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0
		}`

	responseBodyJson :=
		`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-5",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hi! I am Jane.",
						"tool_calls": null,
						"function_call": { "name": "", "arguments": "" },
						"refusal": "",
						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
						"annotations": null
					},
					"finish_reason": "stop",
					"logprobs": { "content": null, "refusal": null }
				}
			],
			"usage": {
				"prompt_tokens": 15,
				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
				"completion_tokens": 30,
				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
				"total_tokens": 45
			},
			"system_fingerprint": "",
			"service_tier": ""
		}`

	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
		InScenario("First Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "application/json").
				WithBody(responseBodyJson),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(requestStub)
	assert.NoError(t, err)

	messageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "Hello Jane!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, messageId)

	replyMessageId, reply, replyerName, err := agent.Reply(ctx, messageId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, replyMessageId)
	assert.Equal(t, "Hi! I am Jane.", reply)
	assert.Equal(t, "Jane", replyerName)

	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyRequestStub)

}

func Test_Agent_should_call_llm_for_the_followup_conversation(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "aganet_should_call_llm_for_the_followup_conversation.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTest(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello Jane!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0
		}`

	firstResponseBodyJson :=
		`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-5",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hi! I am Jane.",
						"tool_calls": null,
						"function_call": { "name": "", "arguments": "" },
						"refusal": "",
						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
						"annotations": null
					},
					"finish_reason": "stop",
					"logprobs": { "content": null, "refusal": null }
				}
			],
			"usage": {
				"prompt_tokens": 15,
				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
				"completion_tokens": 30,
				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
				"total_tokens": 45
			},
			"system_fingerprint": "",
			"service_tier": ""
		}`

	firstRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(firstRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "application/json").
				WithBody(firstResponseBodyJson),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(firstRequestStub)
	assert.NoError(t, err)

	firstMessageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "Hello Jane!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, firstMessageId)

	firstReplyMessageId, firstReply, firstReplyerName, err := agent.Reply(ctx, firstMessageId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, firstReplyMessageId)
	assert.Equal(t, "Hi! I am Jane.", firstReply)
	assert.Equal(t, "Jane", firstReplyerName)

	secondRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello Jane!", "name": "Jim"},
				{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"},
				{"role": "user", "content": "How are you?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0
		}`

	secondResponseBodyJson :=
		`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-5",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "I'm fine, thank you!",
						"tool_calls": null,
						"function_call": { "name": "", "arguments": "" },
						"refusal": "",
						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
						"annotations": null
					},
					"finish_reason": "stop",
					"logprobs": { "content": null, "refusal": null }
				}
			],
			"usage": {
				"prompt_tokens": 15,
				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
				"completion_tokens": 30,
				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
				"total_tokens": 45
			},
			"system_fingerprint": "",
			"service_tier": ""
		}`

	secondRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(secondRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs("first-message-received").
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "application/json").
				WithBody(secondResponseBodyJson),
		).
		WillSetStateTo("second-message-received")

	err = wiremockClient.StubFor(secondRequestStub)
	assert.NoError(t, err)

	secondMessageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondMessageId)

	secondReplyMessageId, secondReply, secondReplyerName, err := agent.Reply(ctx, secondMessageId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondReplyMessageId)
	assert.Equal(t, "I'm fine, thank you!", secondReply)
	assert.Equal(t, "Jane", secondReplyerName)

	verifyFirstRequestStub, err := wiremockClient.Verify(firstRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyFirstRequestStub)

	verifySecondRequestStub, err := wiremockClient.Verify(secondRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifySecondRequestStub)

}

func Test_Agent_should_persist_the_conversation_history_for_the_first_message(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "aganet_should_persist_the_conversation_history_for_the_first_message.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTest(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello Jane!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0
		}`

	responseBodyJson :=
		`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-5",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hi! I am Jane.",
						"tool_calls": null,
						"function_call": { "name": "", "arguments": "" },
						"refusal": "",
						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
						"annotations": null
					},
					"finish_reason": "stop",
					"logprobs": { "content": null, "refusal": null }
				}
			],
			"usage": {
				"prompt_tokens": 15,
				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
				"completion_tokens": 30,
				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
				"total_tokens": 45
			},
			"system_fingerprint": "",
			"service_tier": ""
		}`

	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
		InScenario("First Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "application/json").
				WithBody(responseBodyJson),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(requestStub)
	assert.NoError(t, err)

	messageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "Hello Jane!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, messageId)

	replyMessageId, reply, replyerName, err := agent.Reply(ctx, messageId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, replyMessageId)
	assert.Equal(t, "Hi! I am Jane.", reply)
	assert.Equal(t, "Jane", replyerName)

	allTurns, err := teamDb.Queries.GetTurns(ctx)
	assert.NoError(t, err)

	actualUserThoughtTurn := allTurns[0]
	assert.Equal(t, actualUserThoughtTurn.Kind, string(EventKindThought))
	assert.Equal(t, actualUserThoughtTurn.Status, string(TurnStatusCompleted))

	actualUserThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserThoughtMessage.Visibility, string(VisibilityHidden))
	actualUserThoughtMessageJson, err := json.Marshal(actualUserThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserThoughtMessageJson := fmt.Sprintf(`{"name":"Jim","tool_calls":[{"id":"%s","function":{"arguments":"{\"agent_name\":\"Jane\",\"message\":\"Hello Jane!\"}","name":"articulate_to_agent"},"type":"function"}],"role":"assistant"}`, messageId)
	assert.JSONEq(t, expectedUserThoughtMessageJson, string(actualUserThoughtMessageJson))

	actualUserArticulationTurn := allTurns[1]
	assert.Equal(t, actualUserArticulationTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualUserArticulationTurn.Kind, string(EventKindArticulation))
	actualUserArticulationMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserArticulationTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserArticulationMessage.Visibility, string(VisibilityChannel))
	actualUserArticulationMessageJson, err := json.Marshal(actualUserArticulationMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserArticulationMessageJson := `{"name":"Jim","content":"Hello Jane!","role":"user"}`
	assert.JSONEq(t, expectedUserArticulationMessageJson, string(actualUserArticulationMessageJson))

	actualAgentThinkingTurn := allTurns[2]
	assert.Equal(t, actualAgentThinkingTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualAgentThinkingTurn.Kind, string(EventKindThinking))
	actualAgentThinkingLlmResponse, err := teamDb.Queries.GetLlmResponseByTurn(ctx, actualAgentThinkingTurn.ID)
	assert.NoError(t, err)
	actualAgentThinkingMessageJson, err := json.Marshal(actualAgentThinkingLlmResponse.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, responseBodyJson, string(actualAgentThinkingMessageJson))

	actualAgentThoughtTurn := allTurns[3]
	assert.Equal(t, actualAgentThoughtTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualAgentThoughtTurn.Kind, string(EventKindThought))
	actualAgentThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualAgentThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualAgentThoughtMessage.Visibility, string(VisibilityChannel))
	actualAgentThoughtMessageJson, err := json.Marshal(actualAgentThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedAgentThoughtMessageJson := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedAgentThoughtMessageJson, string(actualAgentThoughtMessageJson))

	assert.Len(t, allTurns, 4)

}

func Test_Agent_should_persist_the_conversation_history_for_the_followup_conversation(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "aganet_should_persist_the_conversation_history_for_the_followup_conversation.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLMTest(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello Jane!", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0
		}`

	firstResponseBodyJson :=
		`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-5",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "Hi! I am Jane.",
						"tool_calls": null,
						"function_call": { "name": "", "arguments": "" },
						"refusal": "",
						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
						"annotations": null
					},
					"finish_reason": "stop",
					"logprobs": { "content": null, "refusal": null }
				}
			],
			"usage": {
				"prompt_tokens": 15,
				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
				"completion_tokens": 30,
				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
				"total_tokens": 45
			},
			"system_fingerprint": "",
			"service_tier": ""
		}`

	firstRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(firstRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "application/json").
				WithBody(firstResponseBodyJson),
		).
		WillSetStateTo("first-message-received")

	err = wiremockClient.StubFor(firstRequestStub)
	assert.NoError(t, err)

	firstMessageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "Hello Jane!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, firstMessageId)

	firstReplyMessageId, firstReply, firstReplyerName, err := agent.Reply(ctx, firstMessageId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, firstReplyMessageId)
	assert.Equal(t, "Hi! I am Jane.", firstReply)
	assert.Equal(t, "Jane", firstReplyerName)

	secondRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello Jane!", "name": "Jim"},
				{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"},
				{"role": "user", "content": "How are you?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0
		}`

	secondResponseBodyJson :=
		`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-5",
			"choices": [
				{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "I'm fine, thank you!",
						"tool_calls": null,
						"function_call": { "name": "", "arguments": "" },
						"refusal": "",
						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
						"annotations": null
					},
					"finish_reason": "stop",
					"logprobs": { "content": null, "refusal": null }
				}
			],
			"usage": {
				"prompt_tokens": 15,
				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
				"completion_tokens": 30,
				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
				"total_tokens": 45
			},
			"system_fingerprint": "",
			"service_tier": ""
		}`

	secondRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(secondRequestBodyJson)).
		InScenario("Second Message to Jane").
		WhenScenarioStateIs("first-message-received").
		WillReturnResponse(
			wiremock.NewResponse().
				WithStatus(http.StatusOK).
				WithHeader("Content-Type", "application/json").
				WithBody(secondResponseBodyJson),
		).
		WillSetStateTo("second-message-received")

	err = wiremockClient.StubFor(secondRequestStub)
	assert.NoError(t, err)

	secondMessageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondMessageId)

	secondReplyMessageId, secondReply, secondReplyerName, err := agent.Reply(ctx, secondMessageId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondReplyMessageId)
	assert.Equal(t, "I'm fine, thank you!", secondReply)
	assert.Equal(t, "Jane", secondReplyerName)

	allTurns, err := teamDb.Queries.GetTurns(ctx)
	assert.NoError(t, err)

	actualUserFirstThoughtTurn := allTurns[0]
	assert.Equal(t, actualUserFirstThoughtTurn.Kind, string(EventKindThought))
	assert.Equal(t, actualUserFirstThoughtTurn.Status, string(TurnStatusCompleted))

	actualUserFirstThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserFirstThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserFirstThoughtMessage.Visibility, string(VisibilityHidden))
	actualUserFirstThoughtMessageJson, err := json.Marshal(actualUserFirstThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserFirstThoughtMessageJson := fmt.Sprintf(`{"name":"Jim","tool_calls":[{"id":"%s","function":{"arguments":"{\"agent_name\":\"Jane\",\"message\":\"Hello Jane!\"}","name":"articulate_to_agent"},"type":"function"}],"role":"assistant"}`, firstMessageId)
	assert.JSONEq(t, expectedUserFirstThoughtMessageJson, string(actualUserFirstThoughtMessageJson))

	actualUserFirstArticulationTurn := allTurns[1]
	assert.Equal(t, actualUserFirstArticulationTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualUserFirstArticulationTurn.Kind, string(EventKindArticulation))
	actualUserFirstArticulationMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserFirstArticulationTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserFirstArticulationMessage.Visibility, string(VisibilityChannel))
	actualUserFirstArticulationMessageJson, err := json.Marshal(actualUserFirstArticulationMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserFirstArticulationMessageJson := `{"name":"Jim","content":"Hello Jane!","role":"user"}`
	assert.JSONEq(t, expectedUserFirstArticulationMessageJson, string(actualUserFirstArticulationMessageJson))

	actualAgentFirstThinkingTurn := allTurns[2]
	assert.Equal(t, actualAgentFirstThinkingTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualAgentFirstThinkingTurn.Kind, string(EventKindThinking))
	actualAgentFirstThinkingLlmResponse, err := teamDb.Queries.GetLlmResponseByTurn(ctx, actualAgentFirstThinkingTurn.ID)
	assert.NoError(t, err)
	actualAgentFirstThinkingMessageJson, err := json.Marshal(actualAgentFirstThinkingLlmResponse.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseBodyJson, string(actualAgentFirstThinkingMessageJson))

	actualAgentFirstThoughtTurn := allTurns[3]
	assert.Equal(t, actualAgentFirstThoughtTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualAgentFirstThoughtTurn.Kind, string(EventKindThought))
	actualAgentFirstThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualAgentFirstThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualAgentFirstThoughtMessage.Visibility, string(VisibilityChannel))
	actualAgentFirstThoughtMessageJson, err := json.Marshal(actualAgentFirstThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedAgentFirstThoughtMessageJson := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedAgentFirstThoughtMessageJson, string(actualAgentFirstThoughtMessageJson))

	actualUserSecondThoughtTurn := allTurns[4]
	assert.Equal(t, actualUserSecondThoughtTurn.Kind, string(EventKindThought))
	assert.Equal(t, actualUserSecondThoughtTurn.Status, string(TurnStatusCompleted))

	actualUserSecondThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserSecondThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserSecondThoughtMessage.Visibility, string(VisibilityHidden))
	actualUserSecondThoughtMessageJson, err := json.Marshal(actualUserSecondThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserSecondThoughtMessageJson := fmt.Sprintf(`{"name":"Jim","tool_calls":[{"id":"%s","function":{"arguments":"{\"agent_name\":\"Jane\",\"message\":\"How are you?\"}","name":"articulate_to_agent"},"type":"function"}],"role":"assistant"}`, secondMessageId)
	assert.JSONEq(t, expectedUserSecondThoughtMessageJson, string(actualUserSecondThoughtMessageJson))

	actualUserSecondArticulationTurn := allTurns[5]
	assert.Equal(t, actualUserFirstArticulationTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualUserSecondArticulationTurn.Kind, string(EventKindArticulation))
	actualUserSecondArticulationMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserSecondArticulationTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserSecondArticulationMessage.Visibility, string(VisibilityChannel))
	actualUserSecondArticulationMessageJson, err := json.Marshal(actualUserSecondArticulationMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserSecondArticulationMessageJson := `{"name":"Jim","content":"How are you?","role":"user"}`
	assert.JSONEq(t, expectedUserSecondArticulationMessageJson, string(actualUserSecondArticulationMessageJson))

	actualAgentSecondThinkingTurn := allTurns[6]
	assert.Equal(t, actualAgentFirstThinkingTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualAgentSecondThinkingTurn.Kind, string(EventKindThinking))
	actualAgentSecondThinkingLlmResponse, err := teamDb.Queries.GetLlmResponseByTurn(ctx, actualAgentSecondThinkingTurn.ID)
	assert.NoError(t, err)
	actualAgentSecondThinkingMessageJson, err := json.Marshal(actualAgentSecondThinkingLlmResponse.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseBodyJson, string(actualAgentSecondThinkingMessageJson))

	actualAgentSecondThoughtTurn := allTurns[7]
	assert.Equal(t, actualAgentSecondThoughtTurn.Status, string(TurnStatusCompleted))
	assert.Equal(t, actualAgentSecondThoughtTurn.Kind, string(EventKindThought))
	actualAgentSecondThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualAgentSecondThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualAgentSecondThoughtMessage.Visibility, string(VisibilityChannel))
	actualAgentSecondThoughtMessageJson, err := json.Marshal(actualAgentSecondThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedAgentSecondThoughtMessageJson := `{"name":"Jane","content":"I'm fine, thank you!","role":"user"}`
	assert.JSONEq(t, expectedAgentSecondThoughtMessageJson, string(actualAgentSecondThoughtMessageJson))

	assert.Len(t, allTurns, 8)

}

// func Test_Dialogue_engine_should_call_llm_with_tools(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	johnTools := []string{"legal_document_full_text_search", "legal_document_index"}
// 	theTeam := makeTeam(johnId, johnTools)

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "legal_document_full_text_search",
// 						"description": "Use this tool to search the full text of a legal document in the repository. You should think of a keyword and use this tool to search the full text of a legal document in the repository. The result contains the IDs of the elements in square brackets.",
// 						"parameters": {
// 							"type": "object",
// 							"properties": {
// 								"keyword": { "type": "string" }
// 							},
// 							"required": ["keyword"]
// 						}
// 					},
// 					"type": "function"
// 				},
// 				{
// 					"function": {
// 						"name": "legal_document_index",
// 						"description": "Use this tool to list all available legal documents in the repository. The result contains the IDs of the documents in square brackets."
// 					},
// 					"type": "function"
// 				},
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	_, err = dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)

// 	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

// func Test_Dialogue_engine_should_acknoledge_the_messages_as_seen(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	historicalMessageAcknowledgement := records.MessageAcknowledgementRecord{
// 		MessageId:  historicalMessage1.MessageId,
// 		DialogueId: historicalMessage1.DialogueId,
// 		Seen:       true,
// 		Ignored:    false,
// 	}
// 	err = messageAcknowledgementsTable.Action(dynamodbClient).Persist(ctx, historicalMessageAcknowledgement)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	currentMessageAcknowledgement := records.MessageAcknowledgementRecord{
// 		MessageId:  currentMessage.MessageId,
// 		DialogueId: currentMessage.DialogueId,
// 		Seen:       false,
// 		Ignored:    false,
// 	}
// 	err = messageAcknowledgementsTable.Action(dynamodbClient).Persist(ctx, currentMessageAcknowledgement)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	_, err = dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)

// 	actualCurrentMessageAcknowledgement := records.MessageAcknowledgementRecord{
// 		MessageId:  currentMessage.MessageId,
// 		DialogueId: currentMessage.DialogueId,
// 		Seen:       false,
// 		Ignored:    false,
// 	}
// 	err = messageAcknowledgementsTable.Action(dynamodbClient).Reconstitute(ctx, &actualCurrentMessageAcknowledgement)
// 	assert.NoError(t, err)
// 	assert.True(t, actualCurrentMessageAcknowledgement.Seen)
// 	assert.False(t, actualCurrentMessageAcknowledgement.Ignored)

// 	actualDialogueMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  currentMessage.MessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actualDialogueMessage)
// 	assert.NoError(t, err)
// 	assert.True(t, actualDialogueMessage.Seen)
// 	assert.False(t, actualDialogueMessage.Ignored)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

// func Test_Dialogue_engine_should_persist_llm_response(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 1)
// 	newMessageId := newMessageIds[0]

// 	startOfChecking := time.Now()
// 	startOfChecking = startOfChecking.Add(time.Second)

// 	actaulNewMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  newMessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actaulNewMessage)
// 	assert.NoError(t, err)
// 	assert.Equal(t, discourceId, actaulNewMessage.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actaulNewMessage.DialogueId)
// 	assert.Equal(t, jane.Id, actaulNewMessage.AuthorId)
// 	assert.Equal(t, jane.Name, actaulNewMessage.AuthorName)
// 	assert.JSONEq(t, `{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"}`, actaulNewMessage.Message)
// 	assert.False(t, actaulNewMessage.HiddenFromLLM)
// 	assert.WithinRange(t, actaulNewMessage.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	actualLlmResponse := records.LLMResponseRecord{
// 		Id: newMessageId,
// 	}
// 	err = llmResponsesTable.Action(dynamodbClient).Reconstitute(ctx, &actualLlmResponse)
// 	assert.NoError(t, err)
// 	assert.JSONEq(t, responseBodyJson, actualLlmResponse.Message)
// 	assert.Equal(t, jane.DialogueId, actualLlmResponse.DialogueId)
// 	assert.WithinRange(t, actualLlmResponse.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Empty(t, lastNCommandsFromQueue)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

// func Test_Dialogue_engine_should_persist_llm_request(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 1)
// 	newMessageId := newMessageIds[0]

// 	actualLlmRequestS3Key := fmt.Sprintf("dialogue/%s.json", newMessageId)

// 	getObjectInput := &s3.GetObjectInput{
// 		Bucket: &llmRequestsBucket,
// 		Key:    &actualLlmRequestS3Key,
// 	}
// 	actualLlmRequestS3ObjectOutput, err := s3Client.GetObject(
// 		ctx,
// 		getObjectInput,
// 	)
// 	assert.NoError(t, err)

// 	actualLlmRequestString, err := loadS3Body(*actualLlmRequestS3ObjectOutput)
// 	assert.NoError(t, err)
// 	assert.JSONEq(t, requestBodyJson, actualLlmRequestString)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Empty(t, lastNCommandsFromQueue)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

// func Test_Dialogue_engine_should_notify_other_participants_if_llm_response_is_a_stop(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 1)
// 	newMessageId := newMessageIds[0]
// 	assert.NoError(t, err)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Empty(t, lastNCommandsFromQueue)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)

// 	expectedNotificationMessage := `{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"}`
// 	actualNotification := chat.Message{}
// 	err = json.Unmarshal([]byte(*lastNCommandsFromQueue[0].Body), &actualNotification)
// 	assert.NoError(t, err)
// 	assert.JSONEq(t, expectedNotificationMessage, actualNotification.Message)
// 	assert.Equal(t, discourceId, actualNotification.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actualNotification.DialogueId)
// 	assert.Equal(t, jane.Id, actualNotification.AuthorId)
// 	assert.Equal(t, newMessageId, actualNotification.MessageId)
// 	assert.Equal(t, connectionId, actualNotification.ConnectionId)

// }

// func Test_Dialogue_engine_should_loop_if_llm_response_is_a_known_tool_call_and_do_not_notify_other_participants(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Hi Jane!", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	historicalMessage2 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jane.Id,
// 		AuthorName:    jane.Name,
// 		Message:       `{"role": "assistant", "content": "Hi John! How can I help you?", "name": "Jane"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage2)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Tell me the time.", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Second)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	llmRequestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Hi Jane!", "name": "John"},
// 				{"role": "assistant", "content": "Hi John! How can I help you?", "name": "Jane"},
// 				{"role": "user", "content": "Tell me the time.", "name": "John"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	llmResponseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"finish_reason": "tool_calls",
// 					"index": 0,
// 					"logprobs": {
// 						"content": null,
// 						"refusal": null
// 					},
// 					"message": {
// 						"content": "",
// 						"refusal": "",
// 						"role": "assistant",
// 						"annotations": [],
// 						"audio": {
// 							"id": "",
// 							"data": "",
// 							"expires_at": 0,
// 							"transcript": ""
// 						},
// 						"function_call": {
// 							"arguments": "",
// 							"name": ""
// 						},
// 						"tool_calls": [
// 							{
// 								"id": "call_boqJLuyytLw3qB9UOI1A6hyE",
// 								"function": {
// 									"arguments": "{}",
// 									"name": "legal_document_index"
// 								},
// 								"type": "function",
// 								"custom": {
// 									"input": "",
// 									"name": ""
// 								}
// 							}
// 						]
// 					}
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	llmRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(llmRequestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(llmResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(llmRequestStub)
// 	assert.NoError(t, err)

// 	indexToolResponseBodyJson := "index tool response"

// 	indexToolRequestStub := wiremock.Get(wiremock.URLPathEqualTo("/knowledge-repository/tools/index")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(indexToolResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(indexToolRequestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 2)
// 	newMessageId := newMessageIds[0]

// 	startOfChecking := time.Now()
// 	startOfChecking = startOfChecking.Add(time.Second)

// 	actaulNewMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  newMessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actaulNewMessage)
// 	assert.NoError(t, err)
// 	assert.Equal(t, discourceId, actaulNewMessage.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actaulNewMessage.DialogueId)
// 	assert.Equal(t, jane.Id, actaulNewMessage.AuthorId)
// 	assert.Equal(t, jane.Name, actaulNewMessage.AuthorName)
// 	assert.JSONEq(t, `{"role": "assistant", "tool_calls": [{"id": "call_boqJLuyytLw3qB9UOI1A6hyE", "function": {"arguments": "{}",	"name": "legal_document_index"}, "type": "function"}], "name": "Jane"}`, actaulNewMessage.Message)
// 	assert.False(t, actaulNewMessage.HiddenFromLLM)
// 	assert.WithinRange(t, actaulNewMessage.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	actualLlmResponse := records.LLMResponseRecord{
// 		Id: newMessageId,
// 	}
// 	err = llmResponsesTable.Action(dynamodbClient).Reconstitute(ctx, &actualLlmResponse)
// 	assert.NoError(t, err)
// 	assert.JSONEq(t, llmResponseBodyJson, actualLlmResponse.Message)
// 	assert.Equal(t, jane.DialogueId, actualLlmResponse.DialogueId)
// 	assert.WithinRange(t, actualLlmResponse.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	expectedSelfPing := fmt.Sprintf(`{"connectionId":"%s","discourseId":"%s","dialogueId":"%s","messageId":"%s"}`, connectionId, discourceId, jane.DialogueId, newMessageId)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)

// 	actualSelfPing := lastNCommandsFromQueue[0]
// 	assert.JSONEq(t, expectedSelfPing, *actualSelfPing.Body)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// }

// func Test_Dialogue_engine_should_call_tools(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Hi Jane!", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	historicalMessage2 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jane.Id,
// 		AuthorName:    jane.Name,
// 		Message:       `{"role": "assistant", "content": "Hi John! How can I help you?", "name": "Jane"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage2)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Tell me the time.", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Second)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	llmRequestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Hi Jane!", "name": "John"},
// 				{"role": "assistant", "content": "Hi John! How can I help you?", "name": "Jane"},
// 				{"role": "user", "content": "Tell me the time.", "name": "John"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	llmResponseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"finish_reason": "tool_calls",
// 					"index": 0,
// 					"logprobs": {
// 						"content": null,
// 						"refusal": null
// 					},
// 					"message": {
// 						"content": "",
// 						"refusal": "",
// 						"role": "assistant",
// 						"annotations": [],
// 						"audio": {
// 							"id": "",
// 							"data": "",
// 							"expires_at": 0,
// 							"transcript": ""
// 						},
// 						"function_call": {
// 							"arguments": "",
// 							"name": ""
// 						},
// 						"tool_calls": [
// 							{
// 								"id": "call_boqJLuyytLw3qB9UOI1A6hyE",
// 								"function": {
// 									"arguments": "{}",
// 									"name": "legal_document_index"
// 								},
// 								"type": "function",
// 								"custom": {
// 									"input": "",
// 									"name": ""
// 								}
// 							},
// 							{
// 								"id": "call_xoqJLuyytLw3qB9UOI1A6hyE",
// 								"function": {
// 									"arguments": "{ \"keyword\": \"theKeyword\"}",
// 									"name": "legal_document_full_text_search"
// 								},
// 								"type": "function",
// 								"custom": {
// 									"input": "",
// 									"name": ""
// 								}
// 							}
// 						]
// 					}
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	llmRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(llmRequestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(llmResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(llmRequestStub)
// 	assert.NoError(t, err)

// 	indexToolResponseBodyJson := "index tool response"

// 	indexToolRequestStub := wiremock.Get(wiremock.URLPathEqualTo("/knowledge-repository/tools/index")).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithBody(indexToolResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(indexToolRequestStub)
// 	assert.NoError(t, err)

// 	fullTextSearchToolResponseBodyJson := "full text search tool response"

// 	fullTextSearchToolRequestStub := wiremock.Get(wiremock.URLPathEqualTo("/knowledge-repository/tools/fts")).
// 		WithQueryParam("keyword", wiremock.EqualTo("theKeyword")).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithBody(fullTextSearchToolResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(fullTextSearchToolRequestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 3)
// 	newMessageId := newMessageIds[0]

// 	startOfChecking := time.Now()
// 	startOfChecking = startOfChecking.Add(time.Second)

// 	actaulNewMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  newMessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actaulNewMessage)
// 	assert.NoError(t, err)
// 	assert.Equal(t, discourceId, actaulNewMessage.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actaulNewMessage.DialogueId)
// 	assert.Equal(t, jane.Id, actaulNewMessage.AuthorId)
// 	assert.Equal(t, jane.Name, actaulNewMessage.AuthorName)
// 	assert.JSONEq(t, `{"role": "assistant", "tool_calls": [{"id": "call_boqJLuyytLw3qB9UOI1A6hyE", "function": {"arguments": "{}",	"name": "legal_document_index"}, "type": "function"}, {"id": "call_xoqJLuyytLw3qB9UOI1A6hyE", "function": {"arguments": "{ \"keyword\": \"theKeyword\"}",	"name": "legal_document_full_text_search"}, "type": "function"}], "name": "Jane"}`, actaulNewMessage.Message)
// 	assert.False(t, actaulNewMessage.HiddenFromLLM)
// 	assert.WithinRange(t, actaulNewMessage.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	actualLlmResponse := records.LLMResponseRecord{
// 		Id: newMessageId,
// 	}
// 	err = llmResponsesTable.Action(dynamodbClient).Reconstitute(ctx, &actualLlmResponse)
// 	assert.NoError(t, err)
// 	assert.JSONEq(t, llmResponseBodyJson, actualLlmResponse.Message)
// 	assert.Equal(t, jane.DialogueId, actualLlmResponse.DialogueId)
// 	assert.WithinRange(t, actualLlmResponse.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	expectedSelfPing := fmt.Sprintf(`{"connectionId":"%s","discourseId":"%s","dialogueId":"%s","messageId":"%s"}`, connectionId, discourceId, jane.DialogueId, newMessageId)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)

// 	actualSelfPing := lastNCommandsFromQueue[0]
// 	assert.JSONEq(t, expectedSelfPing, *actualSelfPing.Body)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	verifyRequestStub, err := wiremockClient.Verify(indexToolRequestStub.Request(), 1)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	verifyRequestStub, err = wiremockClient.Verify(fullTextSearchToolRequestStub.Request(), 1)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// }

// func Test_Dialogue_engine_should_persist_tool_call_results(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Hi Jane!", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	historicalMessage2 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jane.Id,
// 		AuthorName:    jane.Name,
// 		Message:       `{"role": "assistant", "content": "Hi John! How can I help you?", "name": "Jane"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage2)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Tell me the time.", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Second)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	llmRequestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Hi Jane!", "name": "John"},
// 				{"role": "assistant", "content": "Hi John! How can I help you?", "name": "Jane"},
// 				{"role": "user", "content": "Tell me the time.", "name": "John"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	llmResponseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"finish_reason": "tool_calls",
// 					"index": 0,
// 					"logprobs": {
// 						"content": null,
// 						"refusal": null
// 					},
// 					"message": {
// 						"content": "",
// 						"refusal": "",
// 						"role": "assistant",
// 						"annotations": [],
// 						"audio": {
// 							"id": "",
// 							"data": "",
// 							"expires_at": 0,
// 							"transcript": ""
// 						},
// 						"function_call": {
// 							"arguments": "",
// 							"name": ""
// 						},
// 						"tool_calls": [
// 							{
// 								"id": "call_boqJLuyytLw3qB9UOI1A6hyE",
// 								"function": {
// 									"arguments": "{}",
// 									"name": "legal_document_index"
// 								},
// 								"type": "function",
// 								"custom": {
// 									"input": "",
// 									"name": ""
// 								}
// 							},
// 							{
// 								"id": "call_xoqJLuyytLw3qB9UOI1A6hyE",
// 								"function": {
// 									"arguments": "{ \"keyword\": \"theKeyword\"}",
// 									"name": "legal_document_full_text_search"
// 								},
// 								"type": "function",
// 								"custom": {
// 									"input": "",
// 									"name": ""
// 								}
// 							}
// 						]
// 					}
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	llmRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(llmRequestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(llmResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(llmRequestStub)
// 	assert.NoError(t, err)

// 	indexToolResponseBodyJson := "index tool response"

// 	indexToolRequestStub := wiremock.Get(wiremock.URLPathEqualTo("/knowledge-repository/tools/index")).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithBody(indexToolResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(indexToolRequestStub)
// 	assert.NoError(t, err)

// 	fullTextSearchToolResponseBodyJson := "full text search tool response"

// 	fullTextSearchToolRequestStub := wiremock.Get(wiremock.URLPathEqualTo("/knowledge-repository/tools/fts")).
// 		WithQueryParam("keyword", wiremock.EqualTo("theKeyword")).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithBody(fullTextSearchToolResponseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(fullTextSearchToolRequestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 3)

// 	startOfChecking := time.Now()
// 	startOfChecking = startOfChecking.Add(time.Second)

// 	indexToolMessageId := newMessageIds[1]
// 	actaulIndexToolMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  indexToolMessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actaulIndexToolMessage)
// 	assert.NoError(t, err)
// 	assert.Equal(t, discourceId, actaulIndexToolMessage.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actaulIndexToolMessage.DialogueId)
// 	assert.Equal(t, jane.Id, actaulIndexToolMessage.AuthorId)
// 	assert.Equal(t, jane.Name, actaulIndexToolMessage.AuthorName)
// 	assert.JSONEq(t, `{"content":"index tool response", "role":"tool", "tool_call_id":"call_boqJLuyytLw3qB9UOI1A6hyE"}`, actaulIndexToolMessage.Message)
// 	assert.False(t, actaulIndexToolMessage.HiddenFromLLM)
// 	assert.WithinRange(t, actaulIndexToolMessage.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	fullTextSearchToolMessageId := newMessageIds[2]
// 	actualFullTextSearchToolMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  fullTextSearchToolMessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actualFullTextSearchToolMessage)
// 	assert.NoError(t, err)
// 	assert.Equal(t, discourceId, actualFullTextSearchToolMessage.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actualFullTextSearchToolMessage.DialogueId)
// 	assert.Equal(t, jane.Id, actualFullTextSearchToolMessage.AuthorId)
// 	assert.Equal(t, jane.Name, actualFullTextSearchToolMessage.AuthorName)
// 	assert.JSONEq(t, `{"content":"full text search tool response", "role":"tool", "tool_call_id":"call_xoqJLuyytLw3qB9UOI1A6hyE"}`, actualFullTextSearchToolMessage.Message)
// 	assert.False(t, actualFullTextSearchToolMessage.HiddenFromLLM)
// 	assert.WithinRange(t, actualFullTextSearchToolMessage.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// }

// func Test_Dialogue_engine_should_persist_message_as_hidden_from_llm_if_ignore_tool_call_is_detected_and_do_not_notify_other_participants(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"finish_reason": "tool_calls",
// 					"index": 0,
// 					"logprobs": {
// 						"content": null,
// 						"refusal": null
// 					},
// 					"message": {
// 						"content": "",
// 						"refusal": "",
// 						"role": "assistant",
// 						"annotations": [],
// 						"audio": {
// 							"id": "",
// 							"data": "",
// 							"expires_at": 0,
// 							"transcript": ""
// 						},
// 						"function_call": {
// 							"arguments": "",
// 							"name": ""
// 						},
// 						"tool_calls": [
// 							{
// 								"id": "call_JCCOz2CsdXoPX2u9bswjWUt8",
// 								"function": {
// 									"arguments": "{}",
// 									"name": "ignore_the_current_message"
// 								},
// 								"type": "function",
// 								"custom": {
// 									"input": "",
// 									"name": ""
// 								}
// 							}
// 						]
// 					}
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	newMessageIds, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Len(t, newMessageIds, 1)
// 	newMessageId := newMessageIds[0]
// 	assert.NoError(t, err)

// 	startOfChecking := time.Now()
// 	startOfChecking = startOfChecking.Add(time.Second)

// 	actualCurrentMessageAcknowledgement := records.MessageAcknowledgementRecord{
// 		MessageId:  currentMessage.MessageId,
// 		DialogueId: currentMessage.DialogueId,
// 		Seen:       false,
// 		Ignored:    false,
// 	}
// 	err = messageAcknowledgementsTable.Action(dynamodbClient).Reconstitute(ctx, &actualCurrentMessageAcknowledgement)
// 	assert.NoError(t, err)
// 	assert.True(t, actualCurrentMessageAcknowledgement.Seen)
// 	assert.True(t, actualCurrentMessageAcknowledgement.Ignored)

// 	actualCurrentMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  currentMessage.MessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actualCurrentMessage)
// 	assert.NoError(t, err)
// 	assert.True(t, actualCurrentMessage.Seen)
// 	assert.True(t, actualCurrentMessage.Ignored)

// 	actaulNewMessage := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 		MessageId:  newMessageId,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Reconstitute(ctx, &actaulNewMessage)
// 	assert.NoError(t, err)
// 	assert.Equal(t, discourceId, actaulNewMessage.DiscourseId)
// 	assert.Equal(t, jane.DialogueId, actaulNewMessage.DialogueId)
// 	assert.Equal(t, jane.Id, actaulNewMessage.AuthorId)
// 	assert.Equal(t, jane.Name, actaulNewMessage.AuthorName)
// 	assert.JSONEq(t, `{"role": "assistant", "tool_calls": [{"id": "call_JCCOz2CsdXoPX2u9bswjWUt8", "function": {"arguments": "{}",	"name": "ignore_the_current_message"}, "type": "function"}], "name": "Jane"}`, actaulNewMessage.Message)
// 	assert.True(t, actaulNewMessage.HiddenFromLLM)
// 	assert.WithinRange(t, actaulNewMessage.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	actualLlmResponse := records.LLMResponseRecord{
// 		Id: newMessageId,
// 	}
// 	err = llmResponsesTable.Action(dynamodbClient).Reconstitute(ctx, &actualLlmResponse)
// 	assert.NoError(t, err)
// 	assert.JSONEq(t, responseBodyJson, actualLlmResponse.Message)
// 	assert.Equal(t, jane.DialogueId, actualLlmResponse.DialogueId)
// 	assert.WithinRange(t, actualLlmResponse.CreatedAt.ToTime(), startOfTest, startOfChecking)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Empty(t, lastNCommandsFromQueue)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)
// }

// func Test_Dialogue_engine_should_halt_when_discourse_is_cancelled(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           true,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	newMessageId, err := dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)
// 	assert.Empty(t, newMessageId)

// 	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 0)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	messagesKey := records.DialogueMessageRecord{
// 		DialogueId: jane.DialogueId,
// 	}
// 	actualMessages, _, err := dialogueMessagesTable.Action(dynamodbClient).QueryAsc(ctx, messagesKey, nil, 100)
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, len(actualMessages))

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Empty(t, lastNCommandsFromQueue)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)
// }

// func Test_Dialogue_ignore_message_hidden_from_llm(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	historicalMessage2 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "What is the time?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: true,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage2)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Second)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	_, err = dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)

// 	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

// func Test_Dialogue_engine_should_not_call_llm_if_no_unseen_messages_are_found(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": [
// 				{
// 					"function": {
// 						"name": "ignore_the_current_message",
// 						"description": "Call this to ignore the current message if it is not addressed to you."
// 					},
// 					"type": "function"
// 				}
// 			]
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	_, err = dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)

// 	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 0)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)
// }

// func Test_Dialogue_engine_should_not_include_ignore_tool_call_in_the_llm_request_if_the_agent_is_mentioned(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi @Jane! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithoutIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi @Jane! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": []
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	_, err = dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)

// 	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

// func Test_Dialogue_engine_should_not_include_ignore_tool_call_in_the_llm_request_if_all_agents_is_mentioned(t *testing.T) {
// 	var err error
// 	defer wiremockClient.Reset()

// 	ctx := context.TODO()

// 	startOfTest := time.Now()
// 	startOfTest = startOfTest.Add(-time.Second)

// 	johnId := ulid.Make().String()
// 	theTeam := makeTeam(johnId, []string{})

// 	clientGeneratedId := uuid.New().String()
// 	teamCreated, err := teamCreator.CreateTeam(ctx, clientGeneratedId, theTeam, logger)
// 	assert.NoError(t, err)

// 	discourceId := ulid.Make().String()
// 	discourceCreatedAt := startOfTest.Add(-time.Hour)
// 	discourceCreatedAtZulu := zulu.DateTimeMillisFromTime(discourceCreatedAt)
// 	discourse := records.DiscourseRecord{
// 		Id:                    discourceId,
// 		InitialDialogueId:     teamCreated.Initiator.DialogueId,
// 		InitialConversationId: teamCreated.ConversationId,
// 		CreatedAt:             discourceCreatedAtZulu,
// 		IsCancelled:           false,
// 	}
// 	err = discoursesTable.Action(dynamodbClient).Persist(ctx, discourse)
// 	assert.NoError(t, err)

// 	// we are looking at the conversation from the perspective of the Jane (member 0).
// 	john := teamCreated.Initiator
// 	jane := teamCreated.Members[0]
// 	jim := teamCreated.Members[1]

// 	historicalMessage1 := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      john.Id,
// 		AuthorName:    john.Name,
// 		Message:       `{"role": "user", "content": "Who are you?", "name": "John"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Hour)),
// 		Seen:          true,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, historicalMessage1)
// 	assert.NoError(t, err)

// 	currentMessage := records.DialogueMessageRecord{
// 		DialogueId:    jane.DialogueId,
// 		MessageId:     ulid.Make().String(),
// 		AuthorId:      jim.Id,
// 		AuthorName:    jim.Name,
// 		Message:       `{"role": "assistant", "content": "Hi @all! I am Jim.", "name": "Jim"}`,
// 		DiscourseId:   discourceId,
// 		HiddenFromLLM: false,
// 		CreatedAt:     zulu.DateTimeMillisFromTime(startOfTest.Add(-time.Minute)),
// 		Seen:          false,
// 		Ignored:       false,
// 	}
// 	err = dialogueMessagesTable.Action(dynamodbClient).Persist(ctx, currentMessage)
// 	assert.NoError(t, err)

// 	connectionId := ulid.Make().String()
// 	ping := chat.DialoguePing{
// 		ConnectionId: connectionId,
// 		DiscourseId:  discourceId,
// 		DialogueId:   jane.DialogueId,
// 		MessageId:    currentMessage.MessageId,
// 	}

// 	janesSystemMessageJsonString, err := makeJanesSystemMessageWithoutIgnoreJsonString()
// 	assert.NoError(t, err)

// 	requestBodyJson := fmt.Sprintf(
// 		`{
// 			"model": "gpt-5",
// 			"messages": [
// 				{"role": "system", "content": %s},
// 				{"role": "user", "content": "Who are you?", "name": "John"},
// 				{"role": "assistant", "content": "Hi @all! I am Jim.", "name": "Jim"}
// 			],
// 			"n": 1,
// 			"temperature": 1.0,
// 			"parallel_tool_calls": false,
// 			"tools": []
// 		}`, janesSystemMessageJsonString)

// 	responseBodyJson :=
// 		`{
// 			"id": "chatcmpl-123",
// 			"object": "chat.completion",
// 			"created": 1677652288,
// 			"model": "gpt-5",
// 			"choices": [
// 				{
// 					"index": 0,
// 					"message": {
// 						"role": "assistant",
// 						"content": "Hi! I am Jane.",
// 						"tool_calls": null,
// 						"function_call": { "name": "", "arguments": "" },
// 						"refusal": "",
// 						"audio": { "id": "", "data": "", "transcript": "", "expires_at": 0 },
// 						"annotations": null
// 					},
// 					"finish_reason": "stop",
// 					"logprobs": { "content": null, "refusal": null }
// 				}
// 			],
// 			"usage": {
// 				"prompt_tokens": 15,
// 				"prompt_tokens_details": {"cached_tokens":0,"audio_tokens":0},
// 				"completion_tokens": 30,
// 				"completion_tokens_details": {"accepted_prediction_tokens":0,"rejected_prediction_tokens":0,"reasoning_tokens":0,"audio_tokens":0},
// 				"total_tokens": 45
// 			},
// 			"system_fingerprint": "",
// 			"service_tier": ""
// 		}`

// 	requestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
// 		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
// 		WithBodyPattern(wiremock.EqualToJson(requestBodyJson)).
// 		WillReturnResponse(
// 			wiremock.NewResponse().
// 				WithStatus(http.StatusOK).
// 				WithHeader("Content-Type", "application/json").
// 				WithBody(responseBodyJson),
// 		)
// 	err = wiremockClient.StubFor(requestStub)
// 	assert.NoError(t, err)

// 	_, err = dialogueEngine.Converse(ctx, ping, logger)
// 	assert.NoError(t, err)

// 	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
// 	assert.NoError(t, err)
// 	assert.True(t, verifyRequestStub)

// 	lastNCommandsFromQueue, err := queue.GetLastNCommands(ctx, sqsClient, aiDialoguesQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 0)

// 	lastNCommandsFromQueue, err = queue.GetLastNCommands(ctx, sqsClient, conversationsQueueUrl, 9)
// 	assert.NoError(t, err)
// 	assert.Len(t, lastNCommandsFromQueue, 1)
// }

func makeTeamForCallLLMTest(ctx context.Context, teamDb *TeamDb) (err error) {
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
		StreamMode:  false,
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
		StreamMode:  false,
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
		StreamMode:  false,
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
		StreamMode:  false,
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
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	return nil
}
