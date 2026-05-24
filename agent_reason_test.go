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
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
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

	reply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, 1, len(reply.replyStepIds))
	actualReplyStepId := reply.replyStepIds[0]

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
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
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

	firstReply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(firstReply.replyStepIds))
	actualFirstReplyStepId := firstReply.replyStepIds[0]

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
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"},
				{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"},
				{"role": "user", "content": "Jane! How are you?", "name": "Jim"}
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

	secondReply, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, secondReply)
	assert.Equal(t, 1, len(secondReply.replyStepIds))
	actualSecondReplyStepId := secondReply.replyStepIds[0]

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
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
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

	reply, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, reply)
	assert.Equal(t, 1, len(reply.replyStepIds))

	allSteps, err := teamDb.Queries.GetSteps(ctx)
	assert.NoError(t, err)

	actualStep1 := allSteps[0]
	assert.Equal(t, actualStep1.Kind, string(EventKindMentioning))

	actualStep2 := allSteps[1]
	assert.Equal(t, actualStep2.Kind, string(EventKindObserving))
	actualMessage1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep2.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage1.Visibility, string(VisibilityChannel))
	actualMessage1Json, err := json.Marshal(actualMessage1.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage1Json := fmt.Sprintf(`{"name":"Jim","content":"Jane! Hello!","role":"user"}`)
	assert.JSONEq(t, expectedMessage1Json, string(actualMessage1Json))

	actualStep3 := allSteps[2]
	assert.Equal(t, actualStep3.Kind, string(EventKindReasoning))
	actualLlmResponse1, err := teamDb.Queries.GetLlmResponseByStep(ctx, actualStep3.ID)
	assert.NoError(t, err)
	actualLlmResponse1Json, err := json.Marshal(actualLlmResponse1.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, responseBodyJson, string(actualLlmResponse1Json))

	actualStep4 := allSteps[3]
	assert.Equal(t, actualStep4.Kind, string(EventKindActing))
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
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"}
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

	secondRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Jane! Hello!", "name": "Jim"},
				{"role": "assistant", "content": "Hi! I am Jane.", "name": "Jane"},
				{"role": "user", "content": "Jane! How are you?", "name": "Jim"}
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

	allSteps, err := teamDb.Queries.GetSteps(ctx)
	assert.NoError(t, err)

	actualStep1 := allSteps[0]
	assert.Equal(t, actualStep1.Kind, string(EventKindMentioning))

	actualStep2 := allSteps[1]
	assert.Equal(t, actualStep2.Kind, string(EventKindObserving))
	actualMessage1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep2.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage1.Visibility, string(VisibilityChannel))
	actualMessage1Json, err := json.Marshal(actualMessage1.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage1Json := fmt.Sprintf(`{"name":"Jim","content":"Jane! Hello!","role":"user"}`)
	assert.JSONEq(t, expectedMessage1Json, string(actualMessage1Json))

	actualStep3 := allSteps[2]
	assert.Equal(t, actualStep3.Kind, string(EventKindReasoning))
	actualLlmResponse1, err := teamDb.Queries.GetLlmResponseByStep(ctx, actualStep3.ID)
	assert.NoError(t, err)
	actualLlmResponse1Json, err := json.Marshal(actualLlmResponse1.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseBodyJson, string(actualLlmResponse1Json))

	actualStep4 := allSteps[3]
	assert.Equal(t, actualStep4.Kind, string(EventKindActing))
	actualReplying1, err := teamDb.Queries.GetMessageByStep(ctx, actualStep4.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying1.Visibility, string(VisibilityChannel))
	actualReplying1Json, err := json.Marshal(actualReplying1.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying1Json := `{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`
	assert.JSONEq(t, expectedReplying1Json, string(actualReplying1Json))

	actualStep5 := allSteps[4]
	assert.Equal(t, actualStep5.Kind, string(EventKindMentioning))

	actualStep6 := allSteps[5]
	assert.Equal(t, actualStep6.Kind, string(EventKindObserving))
	actualMessage2, err := teamDb.Queries.GetMessageByStep(ctx, actualStep6.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMessage2.Visibility, string(VisibilityChannel))
	actualMessage2Json, err := json.Marshal(actualMessage2.OpenaiMessage)
	assert.NoError(t, err)
	expectedMessage2Json := fmt.Sprintf(`{"name":"Jim","content":"Jane! How are you?","role":"user"}`)
	assert.JSONEq(t, expectedMessage2Json, string(actualMessage2Json))

	actualStep7 := allSteps[6]
	assert.Equal(t, actualStep7.Kind, string(EventKindReasoning))
	actualLlmResponse2, err := teamDb.Queries.GetLlmResponseByStep(ctx, actualStep7.ID)
	assert.NoError(t, err)
	actualLlmResponse2Json, err := json.Marshal(actualLlmResponse2.OpenaiResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseBodyJson, string(actualLlmResponse2Json))

	actualStep8 := allSteps[7]
	assert.Equal(t, actualStep8.Kind, string(EventKindActing))
	actualReplying2, err := teamDb.Queries.GetMessageByStep(ctx, actualStep8.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualReplying2.Visibility, string(VisibilityChannel))
	actualReplying2Json, err := json.Marshal(actualReplying2.OpenaiMessage)
	assert.NoError(t, err)
	expectedReplying2Json := `{"name":"Jane","content":"I'm fine, thank you!","role":"user"}`
	assert.JSONEq(t, expectedReplying2Json, string(actualReplying2Json))

	assert.Len(t, allSteps, 8)

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
