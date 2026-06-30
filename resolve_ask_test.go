package OpenTeam

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Ask_should_create_ask_execution(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Test_Ask_should_create_ask_execution.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

	team := Team{
		ConversationHistoryDb: teamDb,
	}

	messageId, err := team.Ask(ctx, "King", consultingRoom, kingsInquery, testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, messageId)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualMessageRecord, err := teamDb.Queries.GetMessage(ctx, messageId)
	assert.NoError(t, err)

	actualAskExecutionRecord, err := teamDb.Queries.GetExecution(ctx, actualMessageRecord.ExecutionID)
	assert.NoError(t, err)
	assert.Equal(t, actualAskExecutionRecord.Kind, "ask")
	assert.Equal(t, actualAskExecutionRecord.Status, "open")
	assert.WithinRange(t, actualAskExecutionRecord.CreatedAt, startOfTest, startOfChecking)
}

func Test_Ask_should_create_the_message(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Test_Ask_should_create_the_message.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

	team := Team{
		ConversationHistoryDb: teamDb,
	}

	messageId, err := team.Ask(ctx, "King", consultingRoom, kingsInquery, testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, messageId)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualMessageRecord, err := teamDb.Queries.GetMessage(ctx, messageId)
	assert.NoError(t, err)
	assert.Equal(t, actualMessageRecord.ChannelName, consultingRoom)
	assert.Equal(t, actualMessageRecord.Visibility, string(VisibilityChannel))
	assert.Equal(t, actualMessageRecord.RoleID, "king-at-consulting-room")
	assert.WithinRange(t, actualMessageRecord.CreatedAt, startOfTest, startOfChecking)

	expectedOpenAiMessage := fmt.Sprintf(`{"name":"King","content":"%s","role":"user"}`, kingsInquery)
	assert.JSONEq(t, expectedOpenAiMessage, string(actualMessageRecord.OpenaiMessage))

}

func Test_Ask_should_create_mention_executions_for_each_member_of_the_channel(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Test_Ask_should_create_mention_executions_for_each_member_of_the_channel.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

	team := Team{
		ConversationHistoryDb: teamDb,
	}

	messageId, err := team.Ask(ctx, "King", consultingRoom, kingsInquery, testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, messageId)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualMessageRecord, err := teamDb.Queries.GetMessage(ctx, messageId)
	assert.NoError(t, err)

	actualAskExecutionRecord, err := teamDb.Queries.GetExecution(ctx, actualMessageRecord.ExecutionID)
	assert.NoError(t, err)

	actualMentionExecutions, err := teamDb.Queries.GetChildExecutions(ctx, actualAskExecutionRecord.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(actualMentionExecutions), 1)

	actualMentionExecution := actualMentionExecutions[0]
	assert.Equal(t, actualMentionExecution.Kind, "mention")
	assert.Equal(t, actualMentionExecution.Status, "open")
	assert.WithinRange(t, actualMentionExecution.CreatedAt, startOfTest, startOfChecking)

}
