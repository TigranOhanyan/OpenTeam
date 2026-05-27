package OpenTeam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"net/http"
	"testing"
	"time"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/openai/openai-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/wiremock/go-wiremock"
)

func Test_Agent_should_call_llm_when_reasoning(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_when_reasoning.db", testLogger)
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
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	askingStepId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, askingStepId)
	reply, err := agent.Run(ctx, askingStepId, testLogger)
	assert.NotNil(t, reply)
	assert.Equal(t, 1, len(reply.ReplyStepIds))
	actualReplyStepId := reply.ReplyStepIds[0]

	actualReplyMessage, err := teamDb.Queries.GetMessageByStep(ctx, actualReplyStepId)
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

func Test_Agent_should_call_llm_for_the_followup_conversation_when_reasoning(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_for_the_followup_conversation_when_reasoning.db", testLogger)
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
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	firstAskingStepId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, firstAskingStepId)
	firstReply, err := agent.Run(ctx, firstAskingStepId, testLogger)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(firstReply.ReplyStepIds))
	actualFirstReplyStepId := firstReply.ReplyStepIds[0]

	actualFirstReplyMessage, err := teamDb.Queries.GetMessageByStep(ctx, actualFirstReplyStepId)
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
				{"role": "user", "content": "Hello!", "name": "Jim"},
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

	secondAskingStepId, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, secondAskingStepId)
	secondReply, err := agent.Run(ctx, secondAskingStepId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondReply)
	assert.Equal(t, 1, len(secondReply.ReplyStepIds))
	actualSecondReplyStepId := secondReply.ReplyStepIds[0]

	actualSecondReplyMessage, err := teamDb.Queries.GetMessageByStep(ctx, actualSecondReplyStepId)
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

func Test_Agent_should_persist_the_conversation_history_for_the_first_message_when_reasoning(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_first_message_when_reasoning.db", testLogger)
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
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	askingStepId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, askingStepId)
	reply, err := agent.Run(ctx, askingStepId, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, 1, len(reply.ReplyStepIds))

	allSteps, err := teamDb.Queries.GetSteps(ctx)
	assert.NoError(t, err)

	actualRunRecords, err := teamDb.Queries.GetRunsBySourceStep(ctx, askingStepId)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualRunRecords))
	actualRunId := actualRunRecords[0].ID

	actualStep1 := allSteps[0]
	assert.Equal(t, actualStep1.Kind, string(EventKindAsking))
	assert.Nil(t, actualStep1.RunID)
	actualMessage1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep1.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage1.Visibility, string(VisibilityChannel))
	actualMessage1Json, err := json.Marshal(actualMessage1.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage1Json := fmt.Sprintf(`{"name":"Jim","content":"Hello!","role":"user"}`)
	assert.JSONEq(t, expectedMessage1Json, string(actualMessage1Json))

	actualStep2 := allSteps[1]
	assert.Equal(t, actualStep2.Kind, string(EventKindObserving))
	assert.Equal(t, actualStep2.RunID, actualRunId)

	actualStep3 := allSteps[2]
	assert.Equal(t, actualStep3.Kind, string(EventKindReasoning))
	assert.Equal(t, actualStep3.RunID, actualRunId)
	actualLlmResponse1, err := teamDb.Queries.GetLlmResponseByStep(ctx, actualStep3.ID)
	assert.NoError(t, err)
	actualLlmResponse1Json, err := json.Marshal(actualLlmResponse1.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, responseBodyJson, string(actualLlmResponse1Json))

	actualStep4 := allSteps[3]
	assert.Equal(t, actualStep4.Kind, string(EventKindActing))
	assert.Equal(t, actualStep4.RunID, actualRunId)
	actualReplying1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep4.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying1.Visibility, string(VisibilityChannel))
	actualReplying1Json, err := json.Marshal(actualReplying1.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying1Json := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedReplying1Json, string(actualReplying1Json))

	assert.Len(t, allSteps, 4)

}

func Test_Agent_should_persist_the_conversation_history_for_the_followup_conversation_when_reasoning(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_followup_conversation_when_reasoning.db", testLogger)
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
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	firstAskingStepId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, firstAskingStepId)
	_, err = agent.Run(ctx, firstAskingStepId, testLogger)
	assert.NoError(t, err)

	secondRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello!", "name": "Jim"},
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

	secondAskingStepId, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, secondAskingStepId)
	_, err = agent.Run(ctx, secondAskingStepId, testLogger)
	assert.NoError(t, err)

	actualFirstRunRecords, err := teamDb.Queries.GetRunsBySourceStep(ctx, firstAskingStepId)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualFirstRunRecords))
	actualFirstRunId := actualFirstRunRecords[0].ID

	actualSecondRunRecords, err := teamDb.Queries.GetRunsBySourceStep(ctx, secondAskingStepId)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualSecondRunRecords))
	actualSecondRunId := actualSecondRunRecords[0].ID

	allSteps, err := teamDb.Queries.GetSteps(ctx)
	assert.NoError(t, err)

	actualStep1 := allSteps[0]
	assert.Equal(t, actualStep1.Kind, string(EventKindAsking))
	assert.Nil(t, actualStep1.RunID)
	actualMessage1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep1.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage1.Visibility, string(VisibilityChannel))
	actualMessage1Json, err := json.Marshal(actualMessage1.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage1Json := fmt.Sprintf(`{"name":"Jim","content":"Hello!","role":"user"}`)
	assert.JSONEq(t, expectedMessage1Json, string(actualMessage1Json))

	actualStep2 := allSteps[1]
	assert.Equal(t, actualStep2.Kind, string(EventKindObserving))
	assert.Equal(t, actualStep2.RunID, actualFirstRunId)

	actualStep3 := allSteps[2]
	assert.Equal(t, actualStep3.Kind, string(EventKindReasoning))
	assert.Equal(t, actualStep3.RunID, actualFirstRunId)
	actualLlmResponse1, err := teamDb.Queries.GetLlmResponseByStep(ctx, actualStep3.ID)
	assert.NoError(t, err)
	actualLlmResponse1Json, err := json.Marshal(actualLlmResponse1.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseBodyJson, string(actualLlmResponse1Json))

	actualStep4 := allSteps[3]
	assert.Equal(t, actualStep4.Kind, string(EventKindActing))
	assert.Equal(t, actualStep4.RunID, actualFirstRunId)
	actualReplying1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep4.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying1.Visibility, string(VisibilityChannel))
	actualReplying1Json, err := json.Marshal(actualReplying1.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying1Json := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedReplying1Json, string(actualReplying1Json))

	actualStep5 := allSteps[4]
	assert.Equal(t, actualStep5.Kind, string(EventKindAsking))
	assert.Nil(t, actualStep5.RunID)
	actualMessage2, err := teamDb.Queries.GetMessageByStep(ctx, actualStep5.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage2.Visibility, string(VisibilityChannel))
	actualMessage2Json, err := json.Marshal(actualMessage2.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage2Json := fmt.Sprintf(`{"name":"Jim","content":"How are you?","role":"user"}`)
	assert.JSONEq(t, expectedMessage2Json, string(actualMessage2Json))

	actualStep6 := allSteps[5]
	assert.Equal(t, actualStep6.Kind, string(EventKindObserving))
	assert.Equal(t, actualStep6.RunID, actualSecondRunId)

	actualStep7 := allSteps[6]
	assert.Equal(t, actualStep7.Kind, string(EventKindReasoning))
	assert.Equal(t, actualStep7.RunID, actualSecondRunId)
	actualLlmResponse2, err := teamDb.Queries.GetLlmResponseByStep(ctx, actualStep7.ID)
	assert.NoError(t, err)
	actualLlmResponse2Json, err := json.Marshal(actualLlmResponse2.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseBodyJson, string(actualLlmResponse2Json))

	actualStep8 := allSteps[7]
	assert.Equal(t, actualStep8.Kind, string(EventKindActing))
	assert.Equal(t, actualStep8.RunID, actualSecondRunId)
	actualReplying2, err := teamDb.Queries.GetMessageByStep(ctx, actualStep8.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying2.Visibility, string(VisibilityChannel))
	actualReplying2Json, err := json.Marshal(actualReplying2.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying2Json := `{"name":"Jane","content":"I'm fine, thank you!","role":"user"}`
	assert.JSONEq(t, expectedReplying2Json, string(actualReplying2Json))

	assert.Len(t, allSteps, 8)

}

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

	janeLobbyFirstImpression, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "jane-lobby-first-impression",
		RoleID:      janeLobbyRole.ID,
		Instruction: "You are a first impression in the lobby.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
		return err
	}

	janeLobbyDecisionMake, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "jane-lobby-decision-make",
		RoleID:      janeLobbyRole.ID,
		PrevID:      sql.NullString{String: janeLobbyFirstImpression.ID, Valid: true},
		Instruction: "You are a decision maker in the lobby.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
		return err
	}

	_, err = q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "jane-war-room-coordinator",
		RoleID:      janeWarRoomRole.ID,
		PrevID:      sql.NullString{String: janeLobbyDecisionMake.ID, Valid: true},
		Instruction: "You are a coordinator in the war room.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
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

	_, err = q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "john-war-room-expert",
		RoleID:      johnWarRoomRole.ID,
		Instruction: "You are a expert in the war room.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
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

	_, err = q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "jim-the-user",
		RoleID:      jimLobbyRole.ID,
		Instruction: "You are the user.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
		return err
	}

	return nil
}
