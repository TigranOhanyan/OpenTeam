package OpenTeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_MentionResolver_should_create_a_task_execution_for_a_blank_mention(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Test_MentionResolver_should_create_a_task_execution_for_a_blank_mention.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

	team := Team{
		ConversationHistoryDb: teamDb,
	}

	generalResolver := generalResolver{team: &team}
	engine := Engine{ConversationHistoryDb: teamDb}

	messageId, err := team.Ask(ctx, "King", consultingRoom, kingsInquery, testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, messageId)

	report, err := engine.Advance(ctx, testLogger, &generalResolver)
	assert.NoError(t, err)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualMentionExecutionReport := report.Reports[len(report.Reports)-2]
	assert.Equal(t, actualMentionExecutionReport.Status, ExecutionStatusExpanded)

	actualAskExecutionReport := report.Reports[len(report.Reports)-1]
	assert.Equal(t, actualAskExecutionReport.Status, ExecutionStatusSkipped)

	actualAllChildren, err := teamDb.Queries.GetChildExecutions(ctx, actualMentionExecutionReport.ExecutionID)
	assert.NoError(t, err)
	assert.Equal(t, len(actualAllChildren), 1)

	actualTaskExecution := actualAllChildren[0]
	assert.Equal(t, actualTaskExecution.Kind, "task")
	assert.Equal(t, actualTaskExecution.Status, "open")
	assert.WithinRange(t, actualTaskExecution.CreatedAt, startOfTest, startOfChecking)

}

func Test_MentionResolver_should_skip_mention_if_not_all_tasks_are_resolved(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Test_MentionResolver_should_skip_mention_if_not_all_tasks_are_resolved.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

	team := Team{
		ConversationHistoryDb: teamDb,
	}

	generalResolver := generalResolver{team: &team}
	engine := Engine{ConversationHistoryDb: teamDb}

	messageId, err := team.Ask(ctx, "King", consultingRoom, kingsInquery, testLogger)
	assert.NoError(t, err)
	assert.NotEmpty(t, messageId)

	_, err = engine.Advance(ctx, testLogger, &generalResolver)
	assert.NoError(t, err)

	report, err := engine.Advance(ctx, testLogger, &generalResolver)
	assert.NoError(t, err)

	startOfChecking := time.Now()
	startOfChecking = startOfChecking.Add(time.Second)

	actualMentionExecutionReport := report.Reports[len(report.Reports)-2]
	assert.Equal(t, actualMentionExecutionReport.Status, ExecutionStatusSkipped)

	actualAskExecutionReport := report.Reports[len(report.Reports)-1]
	assert.Equal(t, actualAskExecutionReport.Status, ExecutionStatusSkipped)

}

func Test_MentionResolver_should_resolve_mention_if_all_tasks_are_resolved(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Test_MentionResolver_should_resolve_mention_if_all_tasks_are_resolved.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}
