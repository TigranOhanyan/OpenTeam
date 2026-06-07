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
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/stretchr/testify/assert"
	"github.com/wiremock/go-wiremock"
)

func Test_Agent_should_call_llm_with_tools_in_stream_mode(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_with_tools_in_stream_mode.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLWithToolsTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello! What is weather in Yerevan?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true,
			"parallel_tool_calls": true,
			"tools": [
    		    {
    		        "type": "function",
    		        "function": {
    		            "name": "get_current_weather",
						"strict": true,
    		            "description": "Get the current weather in a given location",
    		            "parameters": {
    		                "type": "object",
    		                "properties": {
    		                    "location": {
    		                        "type": "string",
    		                        "description": "The city and state, e.g. San Francisco, CA"
    		                    },
    		                    "unit": {
    		                        "type": "string",
    		                        "enum": [
    		                            "celsius",
    		                            "fahrenheit"
    		                        ]
    		                    }
    		                },
    		                "required": [
    		                    "location"
    		                ]
    		            }
    		        }
    		    }
    		]
		}`

	responseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content": null,"tool_calls": [{"index": 0,"id": "call_yAzMF0nANbOqdFD4hNkezNTM","type": "function","function": {"name": "get_current_weather","arguments": ""}}],"refusal": null},"finish_reason":null}]}`
	responseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"tool_calls": [{"index": 0,"function": {"arguments": "{\"location\":\"Yerevan, Armenia\",\"unit\":\"celsius\"}"}}]},"finish_reason":null}]}`
	responseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
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

	_, err = agent.Ask(ctx, "Jim", "lobby", "Hello! What is weather in Yerevan?", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	verifyRequestStub, err := wiremockClient.Verify(requestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyRequestStub)

}

func Test_Agent_should_persist_the_conversation_history_for_the_first_message_with_tools_in_stream_mode(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_first_message_when_reasoning_in_stream_mode.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLWithToolsTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello! What is weather in Yerevan?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true,
			"parallel_tool_calls": true,
			"tools": [
    		    {
    		        "type": "function",
    		        "function": {
    		            "name": "get_current_weather",
						"strict": true,
    		            "description": "Get the current weather in a given location",
    		            "parameters": {
    		                "type": "object",
    		                "properties": {
    		                    "location": {
    		                        "type": "string",
    		                        "description": "The city and state, e.g. San Francisco, CA"
    		                    },
    		                    "unit": {
    		                        "type": "string",
    		                        "enum": [
    		                            "celsius",
    		                            "fahrenheit"
    		                        ]
    		                    }
    		                },
    		                "required": [
    		                    "location"
    		                ]
    		            }
    		        }
    		    }
    		]
		}`

	responseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content": null,"tool_calls": [{"index": 0,"id": "call_yAzMF0nANbOqdFD4hNkezNTM","type": "function","function": {"name": "get_current_weather","arguments": ""}}],"refusal": null},"finish_reason":null}]}`
	responseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"tool_calls": [{"index": 0,"function": {"arguments": "{\"location\":\"Yerevan, Armenia\",\"unit\":\"celsius\"}"}}]},"finish_reason":null}]}`
	responseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
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

	userMessageId, err := agent.Ask(ctx, "Jim", "lobby", "Hello! What is weather in Yerevan?", testLogger)
	assert.NoError(t, err)

	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	actualRunRecords, err := teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actualRunRecords))

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualReActRunRecord := actualRunRecords[0]
	assert.NoError(t, err)
	assert.Equal(t, actualReActRunRecord.Status, "pending")
	assert.WithinRange(t, actualReActRunRecord.CreatedAt, startOfTest, startOfChecking)

	actualMentionRecord, err := teamDb.Queries.GetMentionByRun(ctx, actualReActRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualMentionRecord.MessageID, userMessageId)
	assert.Equal(t, actualMentionRecord.FromMemberRoleID, "jim-at-lobby")
	assert.Equal(t, actualMentionRecord.ToMemberName, "Jane")
	assert.Equal(t, actualMentionRecord.Message, "Hello! What is weather in Yerevan?")

	actualUserMessageRecord, err := teamDb.Queries.GetMessage(ctx, actualMentionRecord.MessageID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserMessageRecord.Visibility, string(VisibilityChannel))
	actualUserMessageRecordJson, err := json.Marshal(actualUserMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserMessageRecordJson := fmt.Sprintf(`{"name":"Jim","content":"Hello! What is weather in Yerevan?","role":"user"}`)
	assert.JSONEq(t, expectedUserMessageRecordJson, string(actualUserMessageRecordJson))

	allSteps, err := teamDb.Queries.GetStepsByRunId(ctx, actualReActRunRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(allSteps))

	actualCurrentStepRecord := allSteps[0]
	assert.Equal(t, actualCurrentStepRecord.Status, "completed")
	assert.WithinRange(t, actualCurrentStepRecord.CreatedAt, startOfTest, startOfChecking)
	assert.Equal(t, actualCurrentStepRecord.TaskID, "jane-lobby-first-impression")

	actualNextStepRecord := allSteps[1]
	assert.Equal(t, actualNextStepRecord.Status, "pending")
	assert.WithinRange(t, actualNextStepRecord.CreatedAt, startOfTest, startOfChecking)
	assert.Equal(t, actualNextStepRecord.TaskID, "jane-lobby-first-impression")

	actualLlmChunkRecords, err := teamDb.Queries.GetLlmChunkResponseByStep(ctx, actualCurrentStepRecord.ID)
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

	actualAgentMessageRecords, err := teamDb.Queries.GetMessageByStep(ctx, actualCurrentStepRecord.ID)
	assert.NoError(t, err)
	assert.Empty(t, actualAgentMessageRecords)

	actualActionRunRecord := actualRunRecords[1]
	assert.NoError(t, err)
	assert.Equal(t, actualActionRunRecord.Status, "pending")
	assert.WithinRange(t, actualActionRunRecord.CreatedAt, startOfTest, startOfChecking)

	actualActionRecord, err := teamDb.Queries.GetActionByRun(ctx, actualActionRunRecord.ID)
	assert.NoError(t, err)
	actualToolCallJson, err := json.Marshal(actualActionRecord.ToolCall)
	assert.NoError(t, err)
	expectedToolCallJson := `{"type":"function", "id":"call_yAzMF0nANbOqdFD4hNkezNTM", "function":{"name":"get_current_weather","arguments":"{\"location\":\"Yerevan, Armenia\",\"unit\":\"celsius\"}"}}`
	assert.JSONEq(t, expectedToolCallJson, string(actualToolCallJson))
	assert.Nil(t, actualActionRecord.ToolResultMessageID)
	assert.Nil(t, actualActionRecord.ToolRequirementMessageID)

}

func Test_Agent_should_call_llm_by_providing_tool_result_in_stream_mode(t *testing.T) {
	var err error
	wiremockClient.Reset()
	// defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_by_providing_tool_result_in_stream_mode.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLWithToolsTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello! What is weather in Yerevan?", "name": "Jim"}
			],
			"n": 1,
			"temperature": 1.0,
  			"stream": true,
			"parallel_tool_calls": true,
			"tools": [
    		    {
    		        "type": "function",
    		        "function": {
    		            "name": "get_current_weather",
						"strict": true,
    		            "description": "Get the current weather in a given location",
    		            "parameters": {
    		                "type": "object",
    		                "properties": {
    		                    "location": {
    		                        "type": "string",
    		                        "description": "The city and state, e.g. San Francisco, CA"
    		                    },
    		                    "unit": {
    		                        "type": "string",
    		                        "enum": [
    		                            "celsius",
    		                            "fahrenheit"
    		                        ]
    		                    }
    		                },
    		                "required": [
    		                    "location"
    		                ]
    		            }
    		        }
    		    }
    		]
		}`

	firstResponseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content": null,"tool_calls": [{"index": 0,"id": "call_yAzMF0nANbOqdFD4hNkezNTM","type": "function","function": {"name": "get_current_weather","arguments": ""}}],"refusal": null},"finish_reason":null}]}`
	firstResponseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"tool_calls": [{"index": 0,"function": {"arguments": "{\"location\":\"Yerevan, Armenia\",\"unit\":\"celsius\"}"}}]},"finish_reason":null}]}`
	firstResponseChunk3 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
	firstResponseChunk4 :=
		`[DONE]`

	firstResponseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\ndata: %s\n\n", firstResponseChunk1, firstResponseChunk2, firstResponseChunk3, firstResponseChunk4)

	firstRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(firstRequestBodyJson)).
		InScenario("Provide tool result").
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

	secondRequestBodyJson :=
		`{
			"messages": [
				{
					"content": "Hello! What is weather in Yerevan?",
					"name": "Jim",
					"role": "user"
				},
				{
					"name": "Jane",
					"tool_calls": [
						{
							"id": "call_yAzMF0nANbOqdFD4hNkezNTM",
							"function": {
								"arguments": "{\"location\":\"Yerevan, Armenia\",\"unit\":\"celsius\"}",
								"name": "get_current_weather"
							},
							"type": "function"
						}
					],
					"role": "assistant"
				},
				{
					"content": "Yerevan: 35 celsius",
					"tool_call_id": "call_yAzMF0nANbOqdFD4hNkezNTM",
					"role": "tool"
				}
			],
			"model": "gpt-5",
			"n": 1,
			"temperature": 1,
  			"stream": true,
			"parallel_tool_calls": true,
			"tools": [
				{
					"function": {
						"name": "get_current_weather",
						"strict": true,
						"description": "Get the current weather in a given location",
						"parameters": {
							"properties": {
									"location": {
										"description": "The city and state, e.g. San Francisco, CA",
										"type": "string"
									},
									"unit": {
										"enum": [
											"celsius",
											"fahrenheit"
										],
										"type": "string"
									}
							},
							"required": [
								"location"
							],
							"type": "object"
						}
					},
					"type": "function"
				}
			]
		}`

	secondResponseChunk1 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{"role":"assistant","content":"It is 35 celsius in Yerevan."},"finish_reason":null}]}`
	secondResponseChunk2 :=
		`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1677652288,"model":"gpt-5","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	secondResponseChunk3 :=
		`[DONE]`

	secondResponseBodySse := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: %s\n\n", secondResponseChunk1, secondResponseChunk2, secondResponseChunk3)

	secondRequestStub := wiremock.Post(wiremock.URLPathEqualTo("/v1/chat/completions")).
		WithHeader("Content-Type", wiremock.Matching("application/json.*")).
		WithBodyPattern(wiremock.EqualToJson(secondRequestBodyJson)).
		InScenario("Provide tool result").
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

	_, err = agent.Ask(ctx, "Jim", "lobby", "Hello! What is weather in Yerevan?", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	actualRunRecords, err := teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(actualRunRecords))

	actualActionRunRecord := actualRunRecords[1]

	_, _, err = agent.Act(ctx, actualActionRunRecord.ID, "Yerevan: 35 celsius", testLogger)
	assert.NoError(t, err)

	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)

	verifyFirstRequestStub, err := wiremockClient.Verify(firstRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifyFirstRequestStub)

	verifySecondRequestStub, err := wiremockClient.Verify(secondRequestStub.Request(), 1)
	assert.NoError(t, err)
	assert.True(t, verifySecondRequestStub)

}

func test_Agent_should_call_llm_for_the_followup_conversation_with_tools_in_stream_mode(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_call_llm_for_the_followup_conversation_when_reasoning_in_stream_mode.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLWithToolsTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	_, err = agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)
	err = agent.Run(ctx, testLogger)
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

func test_Agent_should_persist_the_conversation_history_for_the_first_message_with_tools_in_stream_mode(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_first_message_when_reasoning_in_stream_mode.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLWithToolsTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	requestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	userMessageId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
	assert.NoError(t, err)

	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)
	actualRunRecords, err := teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualRunRecords))

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualRunRecord := actualRunRecords[0]
	assert.NoError(t, err)
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

	allSteps, err := teamDb.Queries.GetSteps(ctx)
	assert.NoError(t, err)

	actualStep1 := allSteps[0]
	assert.Equal(t, actualStep1.Status, "completed")
	assert.WithinRange(t, actualStep1.CreatedAt, startOfTest, startOfChecking)
	assert.Equal(t, actualStep1.TaskID, "jane-lobby-first-impression")

	actualLlmChunkRecords, err := teamDb.Queries.GetLlmChunkResponseByStep(ctx, actualStep1.ID)
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

	actualAgentMessageRecords, err := teamDb.Queries.GetMessageByStep(ctx, actualStep1.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(actualAgentMessageRecords), 1)
	actualAgentMessageRecord := actualAgentMessageRecords[0]
	assert.Equal(t, actualAgentMessageRecord.Visibility, string(VisibilityChannel))
	actualAgentMessageRecordJson, err := json.Marshal(actualAgentMessageRecord.OpenaiMessage)
	assert.NoError(t, err)
	expectedAgentMessageRecordJson := fmt.Sprintf(`{"name":"Jane","content":"Hi! I am Jane.","role":"user"}`)
	assert.JSONEq(t, expectedAgentMessageRecordJson, string(actualAgentMessageRecordJson))

}

func test_Agent_should_persist_the_conversation_history_for_the_followup_conversation_with_tools_in_stream_mode(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Agent_should_persist_the_conversation_history_for_the_followup_conversation_when_reasoning_in_stream_mode.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForCallLLWithToolsTestInStreamMode(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	firstRequestBodyJson :=
		`{
			"model": "gpt-5",
			"messages": [
				{"role": "user", "content": "Hello!", "name": "Jim"}
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

	firstUserMessageId, err := agent.Ask(ctx, "Jim", "lobby", "Hello!", testLogger)
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

	secondUserMessageId, err := agent.Ask(ctx, "Jim", "lobby", "How are you?", testLogger)
	assert.NoError(t, err)

	err = agent.Run(ctx, testLogger)
	assert.NoError(t, err)
	actualRunRecords, err = teamDb.Queries.GetAllRuns(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(actualRunRecords))

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

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

	actualFirLlmChunkRecords, err := teamDb.Queries.GetLlmChunkResponseByStep(ctx, actualFirstStepRecord.ID)
	assert.NoError(t, err)
	assert.Len(t, actualFirLlmChunkRecords, 3)
	actualFirLlmChunk1Json, err := json.Marshal(actualFirLlmChunkRecords[0].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseChunk1, string(actualFirLlmChunk1Json))
	actualFirLlmChunk2Json, err := json.Marshal(actualFirLlmChunkRecords[1].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseChunk2, string(actualFirLlmChunk2Json))
	actualFirLlmChunk3Json, err := json.Marshal(actualFirLlmChunkRecords[2].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, firstResponseChunk3, string(actualFirLlmChunk3Json))

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

	actualSecondLlmChunkRecords, err := teamDb.Queries.GetLlmChunkResponseByStep(ctx, actualSecondStepRecord.ID)
	assert.NoError(t, err)
	assert.Len(t, actualSecondLlmChunkRecords, 2)
	actualSecondLlmChunk1Json, err := json.Marshal(actualSecondLlmChunkRecords[0].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseChunk1, string(actualSecondLlmChunk1Json))
	actualSecondLlmChunk2Json, err := json.Marshal(actualSecondLlmChunkRecords[1].OpenaiChunkResponse)
	assert.NoError(t, err)
	assert.JSONEq(t, secondResponseChunk2, string(actualSecondLlmChunk2Json))

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

func makeTeamForCallLLWithToolsTestInStreamMode(ctx context.Context, teamDb *TeamDb) (err error) {
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
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
		return err
	}

	weatherTool := openai.ChatCompletionToolUnionParam{
		OfFunction: &openai.ChatCompletionFunctionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        "get_current_weather",
				Description: param.NewOpt("Get the current weather in a given location"),
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]interface{}{
						"location": map[string]string{
							"type":        "string",
							"description": "The city and state, e.g. San Francisco, CA",
						},
						"unit": map[string]any{
							"type": "string",
							"enum": []string{"celsius", "fahrenheit"},
						},
					},
					"required": []string{"location"},
				},
				Strict: param.NewOpt(true),
			},
			Type: "function",
		},
	}

	weatherToolJson, err := json.Marshal(weatherTool)
	if err != nil {
		testLogger.Error("failed to marshal weather tool", zap.Error(err))
		return err
	}

	_, err = q.CreateTool(ctx, entities.CreateToolParams{
		ID:     "jane-lobby-first-impression-weather-tool",
		TaskID: janeLobbyFirstImpression.ID,
		Tool:   weatherToolJson,
	})
	if err != nil {
		testLogger.Error("failed to create tool", zap.Error(err))
		return err
	}

	janeLobbyDecisionMake, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "jane-lobby-decision-make",
		RoleID:      janeLobbyRole.ID,
		PrevID:      sql.NullString{String: janeLobbyFirstImpression.ID, Valid: true},
		Instruction: "You are a decision maker in the lobby.",
		Model:       "gpt-5",
		StreamMode:  true,
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
		StreamMode:  true,
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
		StreamMode:  true,
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
		StreamMode:  true,
	})
	if err != nil {
		testLogger.Error("failed to create task", zap.Error(err))
		return err
	}

	return nil
}
