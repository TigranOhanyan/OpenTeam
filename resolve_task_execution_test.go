package OpenTeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_TaskExecutionResolver_should_create_react_loop_for_a_blank_task(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "TaskExecutionResolver_should_create_react_loop_for_a_blank_task.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_TaskExecutionResolver_should_create_react_loop_when_the_previous_react_loop_has_tool_calls(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "TaskExecutionResolver_should_create_react_loop_when_the_previous_react_loop_has_tool_calls.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_TaskExecutionResolver_should_skip_the_task_execution_when_the_last_react_loop_has_not_been_resolved(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "TaskExecutionResolver_should_skip_the_task_execution_when_the_last_react_loop_has_not_been_resolved.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_TaskExecutionResolver_should_close_the_task_execution_when_the_last_react_loop_has_been_resolved_and_the_status_is_reason(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "TaskExecutionResolver_should_close_the_task_execution_when_the_last_react_loop_has_been_resolved_and_the_status_is_reason.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}
