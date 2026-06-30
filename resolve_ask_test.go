package OpenTeam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_Ask_should_create_mention_executions_for_each_member_of_the_channel(t *testing.T) {
	var err error

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Ask_should_create_mention_executions_for_each_member_of_the_channel.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = sherlockHolmesAgency(ctx, teamDb, testLogger)
	assert.NoError(t, err)

}
