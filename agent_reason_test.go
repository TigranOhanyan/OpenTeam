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
	"github.com/stretchr/testify/assert"
	"github.com/wiremock/go-wiremock"
)

func Test_Agent_should_call_llm_when_reasoning(t *testing.T) {
	var err error
	wiremockClient.Reset()
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

	_, err = agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyRequestStub)

}

func Test_Agent_should_call_llm_for_the_followup_conversation_when_reasoning(t *testing.T) {
	var err error
	wiremockClient.Reset()
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

	_, err = agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)
	actualRunRecords, err := teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualRunRecords))

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

	_, err = agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)
	actualRunRecords, err = teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actualRunRecords))

	verifyFirstRequestStub, err := wiremockClient.Verify(firstRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyFirstRequestStub)

	verifySecondRequestStub, err := wiremockClient.Verify(secondRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifySecondRequestStub)

}

func Test_Agent_should_persist_the_conversation_history_for_the_first_message_when_reasoning(t *testing.T) {
	var err error
	wiremockClient.Reset()
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

	userMessageId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)

	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualRunRecords, err := teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualRunRecords))
	actualRunRecord := actualRunRecords[0]
	assert.Equal(t, actualRunRecord.Status, "completed")
	assert.WithinRange(t, actualRunRecord.CreatedAt, startOfTest, startOfChecking)

	actualMentionRecord, err := teamDb.Queries.GetMentionByRun(ctx, actualRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMentionRecord.MessageID, userMessageId)
	assert.Equal(t, actualMentionRecord.FromMemberRoleID, "jim-at-lobby")
	assert.Equal(t, actualMentionRecord.ToMemberName, "Jane")
	assert.Equal(t, actualMentionRecord.Message, "Hello!")

	actualUserMessageRecord, err := teamDb.Queries.GetMessage(ctx, actualMentionRecord.MessageID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserMessageRecord.Visibility, string(VisibilityChannel))
	actualUserMessageRecordJson, err := json.Marshal(actualUserMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserMessageRecordJson := fmt.Sprintf(`{"name":"Jim","content":"Hello!","role":"user"}`)
	assert.JSONEq(t, expectedUserMessageRecordJson, string(actualUserMessageRecordJson))

	allSteps, err := teamDb.Queries.GetStepsByRunId(ctx, actualRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(allSteps))
	actualStepRecord := allSteps[0]
	assert.Equal(t, actualStepRecord.Status, "completed")
	assert.WithinRange(t, actualStepRecord.CreatedAt, startOfTest, startOfChecking)
	assert.Equal(t, actualStepRecord.TaskID, "jane-lobby-first-impression")

	actualAgentMessageRecords, err := teamDb.Queries.GetMessageByStep(ctx, actualStepRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(actualAgentMessageRecords), 1)
	actualAgentMessageRecord := actualAgentMessageRecords[0]
	assert.Equal(t, actualAgentMessageRecord.Visibility, string(VisibilityChannel))
	actualAgentMessageRecordJson, err := json.Marshal(actualAgentMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedAgentMessageRecordJson := fmt.Sprintf(`{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`)
	assert.JSONEq(t, expectedAgentMessageRecordJson, string(actualAgentMessageRecordJson))

}

func Test_Agent_should_persist_the_conversation_history_for_the_followup_conversation_when_reasoning(t *testing.T) {
	var err error
	wiremockClient.Reset()
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

	firstUserMessageId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)

	err = agent.Run(ctx, testLogger)
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

	secondUserMessageId, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualRunRecords, err := teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actualRunRecords))

	actualFirstRunRecord := actualRunRecords[0]
	assert.NoError(t, err)
	assert.Equal(t, actualFirstRunRecord.Status, "completed")
	assert.WithinRange(t, actualFirstRunRecord.CreatedAt, startOfTest, startOfChecking)

	actualFirstMentionRecord, err := teamDb.Queries.GetMentionByRun(ctx, actualFirstRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualFirstMentionRecord.MessageID, firstUserMessageId)
	assert.Equal(t, actualFirstMentionRecord.FromMemberRoleID, "jim-at-lobby")
	assert.Equal(t, actualFirstMentionRecord.ToMemberName, "Jane")
	assert.Equal(t, actualFirstMentionRecord.Message, "Hello!")
	actualFirstUserMessageRecord, err := teamDb.Queries.GetMessage(ctx, actualFirstMentionRecord.MessageID)
	assert.NoError(t, err)
	assert.Equal(t, actualFirstUserMessageRecord.Visibility, string(VisibilityChannel))
	actualFirstUserMessageRecordJson, err := json.Marshal(actualFirstUserMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedFirstUserMessageRecordJson := fmt.Sprintf(`{"name":"Jim","content":"Hello!","role":"user"}`)
	assert.JSONEq(t, expectedFirstUserMessageRecordJson, string(actualFirstUserMessageRecordJson))

	allFirstSteps, err := teamDb.Queries.GetStepsByRunId(ctx, actualFirstRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(allFirstSteps))
	actualFirstStepRecord := allFirstSteps[0]
	assert.Equal(t, actualFirstStepRecord.Status, "completed")
	assert.WithinRange(t, actualFirstStepRecord.CreatedAt, startOfTest, startOfChecking)
	assert.Equal(t, actualFirstStepRecord.TaskID, "jane-lobby-first-impression")

	actualFirstAgentMessageRecords, err := teamDb.Queries.GetMessageByStep(ctx, actualFirstStepRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(actualFirstAgentMessageRecords), 1)
	actualFirstAgentMessageRecord := actualFirstAgentMessageRecords[0]
	assert.Equal(t, actualFirstAgentMessageRecord.Visibility, string(VisibilityChannel))
	actualFirstAgentMessageRecordJson, err := json.Marshal(actualFirstAgentMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedFirstAgentMessageRecordJson := fmt.Sprintf(`{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`)
	assert.JSONEq(t, expectedFirstAgentMessageRecordJson, string(actualFirstAgentMessageRecordJson))

	actualSecondRunRecord := actualRunRecords[1]
	assert.NoError(t, err)
	assert.Equal(t, actualSecondRunRecord.Status, "completed")
	assert.WithinRange(t, actualSecondRunRecord.CreatedAt, startOfTest, startOfChecking)

	actualSecondMentionRecord, err := teamDb.Queries.GetMentionByRun(ctx, actualSecondRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualSecondMentionRecord.MessageID, secondUserMessageId)
	assert.Equal(t, actualSecondMentionRecord.FromMemberRoleID, "jim-at-lobby")
	assert.Equal(t, actualSecondMentionRecord.ToMemberName, "Jane")
	assert.Equal(t, actualSecondMentionRecord.Message, "How are you?")
	actualSecondUserMessageRecord, err := teamDb.Queries.GetMessage(ctx, actualSecondMentionRecord.MessageID)
	assert.NoError(t, err)
	assert.Equal(t, actualSecondUserMessageRecord.Visibility, string(VisibilityChannel))
	actualSecondUserMessageRecordJson, err := json.Marshal(actualSecondUserMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedSecondUserMessageRecordJson := fmt.Sprintf(`{"name":"Jim","content":"How are you?","role":"user"}`)
	assert.JSONEq(t, expectedSecondUserMessageRecordJson, string(actualSecondUserMessageRecordJson))

	allSecondSteps, err := teamDb.Queries.GetStepsByRunId(ctx, actualSecondRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(allSecondSteps))
	actualSecondStepRecord := allSecondSteps[0]
	assert.Equal(t, actualSecondStepRecord.Status, "completed")
	assert.WithinRange(t, actualSecondStepRecord.CreatedAt, startOfTest, startOfChecking)
	assert.Equal(t, actualSecondStepRecord.TaskID, "jane-lobby-first-impression")
	actualSecondAgentMessageRecords, err := teamDb.Queries.GetMessageByStep(ctx, actualSecondStepRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(actualSecondAgentMessageRecords), 1)
	actualSecondAgentMessageRecord := actualSecondAgentMessageRecords[0]
	assert.Equal(t, actualSecondAgentMessageRecord.Visibility, string(VisibilityChannel))
	actualSecondAgentMessageRecordJson, err := json.Marshal(actualSecondAgentMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedSecondAgentMessageRecordJson := fmt.Sprintf(`{"name":"Jane","content":"I'm fine, thank you!","role":"user"}`)
	assert.JSONEq(t, expectedSecondAgentMessageRecordJson, string(actualSecondAgentMessageRecordJson))

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
