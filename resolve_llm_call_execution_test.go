package OpenTeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ResolveLLMCallBulkExecution_should_provide_llm_call_bulk_response(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ResolveLLMCallBulkExecution_should_provide_llm_call_bulk_response.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}

func Test_ResolveLLMCallChunkExecution_should_provide_llm_call_chunk_response(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "ResolveLLMCallChunkExecution_should_provide_llm_call_chunk_response.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}
