package OpenTeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ReactExecutionResolver_should_create_reason_for_a_blank_react_loop(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ReactExecutionResolver_should_create_reason_for_a_blank_react_loop.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_ReactExecutionResolver_should_create_act_once_the_reason_has_been_resolved_and_the_llm_response_has_tool_calls(t *testing.T) {
	// here you may provide an llm response the way user would do
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ReactExecutionResolver_should_create_act_once_the_reason_has_been_resolved_and_the_llm_response_has_tool_calls.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_ReactExecutionResolver_should_closed_the_react_execution_when_the_reason_has_been_resolved_and_the_llm_response_has_no_tool_calls(t *testing.T) {
	// here you may provide an llm response the way user would do
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ReactExecutionResolver_should_closed_the_react_execution_when_the_reason_has_been_resolved_and_the_llm_response_has_no_tool_calls.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}
