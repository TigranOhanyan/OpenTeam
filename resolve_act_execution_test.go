package OpenTeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ActExecutionResolver_should_resolve_act_execution_if_all_tool_calls_have_been_resolved(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ActExecutionResolver_should_resolve_act_execution.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_ActExecutionResolver_should_skip_act_execution_if_not_all_tool_calls_have_been_resolved(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ActExecutionResolver_should_resolve_act_execution.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)
}
