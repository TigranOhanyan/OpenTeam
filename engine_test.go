package OpenTeam

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"testing"
	"time"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"github.com/stretchr/testify/assert"
)

func Test_Engine_should_resolve_the_execution_if_all_children_are_resolved_and_resolver_is_contracting(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Engine_when_called_consecutively_should_resolve_the_grandparent_execution_if_all_children_and_grandchildren_are_resolved_and_resolver_is_contracting.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeExecutionGraph(ctx, teamDb)
	assert.NoError(t, err)

	_, err = teamDb.Queries.CloseExecution(ctx, "exec31")
	assert.NoError(t, err)
	_, err = teamDb.Queries.CloseExecution(ctx, "exec32")
	assert.NoError(t, err)

	engine := Engine{
		ConversationHistoryDb: teamDb,
	}

	resolver := contractingResolver{}
	executionReports, err := engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(executionReports.Reports))

	startOfCheck := time.Now()
	startOfCheck = startOfCheck.Add(time.Second)

	actualExecutionReport := executionReports.Reports[0]

	assert.Equal(t, actualExecutionReport.Parent.Kind, "mention")
	assert.Equal(t, actualExecutionReport.Parent.Status, "closed")
	assert.Equal(t, actualExecutionReport.Parent.ID, "exec2")
	assert.Equal(t, len(actualExecutionReport.NewlyOpenedChildren), 0)

	actualExecution, err := teamDb.Queries.GetExecution(ctx, "exec2")
	assert.NoError(t, err)
	assert.Equal(t, actualExecutionReport.Parent, actualExecution)

	assert.Equal(t, 1, len(executionReports.OpenExecutionIds))
	assert.Equal(t, executionReports.OpenExecutionIds[0], "exec1")
	assert.False(t, executionReports.IsResolved())
	assert.False(t, executionReports.IsIdle())

}

func Test_Engine_when_called_consecutively_should_resolve_the_grandparent_execution_if_all_children_and_grandchildren_are_resolved_and_resolver_is_contracting(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Engine_when_called_consecutively_should_resolve_the_grandparent_execution_if_all_children_and_grandchildren_are_resolved_and_resolver_is_contracting.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeExecutionGraph(ctx, teamDb)
	assert.NoError(t, err)

	_, err = teamDb.Queries.CloseExecution(ctx, "exec31")
	assert.NoError(t, err)
	_, err = teamDb.Queries.CloseExecution(ctx, "exec32")
	assert.NoError(t, err)

	engine := Engine{
		ConversationHistoryDb: teamDb,
	}

	resolver := contractingResolver{}
	_, err = engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)

	executionReports, err := engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(executionReports.Reports))

	startOfCheck := time.Now()
	startOfCheck = startOfCheck.Add(time.Second)

	actualExecutionReport := executionReports.Reports[0]

	assert.Equal(t, actualExecutionReport.Parent.Kind, "mention")
	assert.Equal(t, actualExecutionReport.Parent.Status, "closed")
	assert.Equal(t, actualExecutionReport.Parent.ID, "exec1")
	assert.Equal(t, len(actualExecutionReport.NewlyOpenedChildren), 0)

	actualExecution, err := teamDb.Queries.GetExecution(ctx, "exec1")
	assert.NoError(t, err)
	assert.Equal(t, actualExecutionReport.Parent, actualExecution)

	assert.Equal(t, 0, len(executionReports.OpenExecutionIds))
	assert.True(t, executionReports.IsResolved())
	assert.False(t, executionReports.IsIdle())
}

func Test_Engine_when_all_executions_are_resolved_should_return_an_empty_report(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Engine_when_called_consecutively_should_resolve_the_grandparent_execution_if_all_children_and_grandchildren_are_resolved_and_resolver_is_contracting.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeExecutionGraph(ctx, teamDb)
	assert.NoError(t, err)

	_, err = teamDb.Queries.CloseExecution(ctx, "exec31")
	assert.NoError(t, err)
	_, err = teamDb.Queries.CloseExecution(ctx, "exec32")
	assert.NoError(t, err)

	engine := Engine{
		ConversationHistoryDb: teamDb,
	}

	resolver := contractingResolver{}
	_, err = engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)

	_, err = engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)

	executionReports, err := engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(executionReports.Reports))

	assert.Equal(t, 0, len(executionReports.OpenExecutionIds))
	assert.True(t, executionReports.IsResolved())
	assert.True(t, executionReports.IsIdle())
}

func Test_Engine_should_spawn_an_execution_if_all_children_are_resolved_and_resolver_is_expanding(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Engine_when_all_executions_are_resolved_should_return_an_empty_report.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeExecutionGraph(ctx, teamDb)
	assert.NoError(t, err)

	_, err = teamDb.Queries.CloseExecution(ctx, "exec31")
	assert.NoError(t, err)
	_, err = teamDb.Queries.CloseExecution(ctx, "exec32")
	assert.NoError(t, err)

	engine := Engine{
		ConversationHistoryDb: teamDb,
	}

	resolver := expandingResolver{}
	executionReports, err := engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(executionReports.Reports))

	startOfCheck := time.Now()
	startOfCheck = startOfCheck.Add(time.Second)

	actualExecutionReport := executionReports.Reports[0]

	assert.Equal(t, actualExecutionReport.Parent.Kind, "mention")
	assert.Equal(t, actualExecutionReport.Parent.Status, "open")
	assert.Equal(t, actualExecutionReport.Parent.ID, "exec2")
	assert.Equal(t, len(actualExecutionReport.NewlyOpenedChildren), 1)

	actualNewExecutionFromReport := actualExecutionReport.NewlyOpenedChildren[0]
	assert.WithinRange(t, actualNewExecutionFromReport.CreatedAt, startOfTest, startOfCheck)
	assert.Equal(t, actualNewExecutionFromReport.ID, "exec2-next")
	assert.Equal(t, actualNewExecutionFromReport.Kind, "mention")
	assert.Equal(t, actualNewExecutionFromReport.Status, "open")

	actualParentExecution, err := teamDb.Queries.GetExecution(ctx, "exec2")
	assert.NoError(t, err)
	assert.Equal(t, actualExecutionReport.Parent, actualParentExecution)

	actualNewExecutionFromDb, err := teamDb.Queries.GetExecution(ctx, "exec2-next")
	assert.NoError(t, err)
	assert.Equal(t, actualNewExecutionFromDb, actualNewExecutionFromReport)

	assert.Equal(t, 3, len(executionReports.OpenExecutionIds))
	expectedOpenExecutionIds := []string{"exec1", "exec2", "exec2-next"}
	assert.ElementsMatch(t, executionReports.OpenExecutionIds, expectedOpenExecutionIds)
	assert.False(t, executionReports.IsResolved())
	assert.False(t, executionReports.IsIdle())
}

func Test_Engine_when_called_consecutively_should_return_empty_report_if_resolver_is_expanding(t *testing.T) {
	var err error
	wiremockClient.Reset()
	defer wiremockClient.Reset()

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "Engine_when_called_consecutively_should_return_empty_report_if_resolver_is_expanding.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeExecutionGraph(ctx, teamDb)
	assert.NoError(t, err)

	_, err = teamDb.Queries.CloseExecution(ctx, "exec31")
	assert.NoError(t, err)
	_, err = teamDb.Queries.CloseExecution(ctx, "exec32")
	assert.NoError(t, err)

	engine := Engine{
		ConversationHistoryDb: teamDb,
	}

	resolver := expandingResolver{}
	_, err = engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)

	executionReports, err := engine.Advance(ctx, testLogger, &resolver)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(executionReports.Reports))

	assert.Equal(t, 3, len(executionReports.OpenExecutionIds))
	expectedOpenExecutionIds := []string{"exec1", "exec2", "exec2-next"}
	assert.ElementsMatch(t, executionReports.OpenExecutionIds, expectedOpenExecutionIds)
	assert.False(t, executionReports.IsResolved())
	assert.True(t, executionReports.IsIdle())
}

type contractingResolver struct{}

func (r *contractingResolver) Resolve(
	ctx context.Context,
	qtx *entities.Queries,
	parent entities.Execution,
	children []entities.Execution,
	logger *zap.Logger,
) (
	executionReport ExecutionReport,
	err error,
) {

	closedExecution, err := qtx.CloseExecution(ctx, parent.ID)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}

	executionReport = ExecutionReport{
		Parent:              closedExecution,
		NewlyOpenedChildren: []entities.Execution{},
	}

	return
}

type expandingResolver struct{}

func (r *expandingResolver) Resolve(
	ctx context.Context,
	qtx *entities.Queries,
	parent entities.Execution,
	children []entities.Execution,
	logger *zap.Logger,
) (
	executionReport ExecutionReport,
	err error,
) {

	newExecId := fmt.Sprintf("%s-next", parent.ID)

	newExecutionParams := entities.CreateExecutionParams{
		ID:   newExecId,
		Kind: parent.Kind,
	}

	newExecution, err := qtx.CreateExecution(ctx, newExecutionParams)
	if err != nil {
		logger.Error("failed to create execution", zap.Error(err))
		return
	}

	newExecutionLinkParams := entities.CreateExecutionLinkParams{
		ParentID: parent.ID,
		ChildID:  newExecution.ID,
	}
	_, err = qtx.CreateExecutionLink(ctx, newExecutionLinkParams)
	if err != nil {
		logger.Error("failed to create execution link", zap.Error(err))
		return
	}
	executionReport = ExecutionReport{
		Parent:              parent,
		NewlyOpenedChildren: []entities.Execution{newExecution},
	}

	return
}

func makeExecutionGraph(ctx context.Context, teamDb *TeamDb) (err error) {
	q := teamDb.Queries

	exec1Params := entities.CreateExecutionParams{
		ID:   "exec1",
		Kind: "mention",
	}
	exec1, err := q.CreateExecution(ctx, exec1Params)
	if err != nil {
		testLogger.Error("failed to create execution", zap.Error(err))
		return err
	}

	exec2Params := entities.CreateExecutionParams{
		ID:   "exec2",
		Kind: "mention",
	}
	exec2, err := q.CreateExecution(ctx, exec2Params)
	if err != nil {
		testLogger.Error("failed to create execution", zap.Error(err))
		return err
	}

	exec1ToExec2Params := entities.CreateExecutionLinkParams{
		ParentID: exec1.ID,
		ChildID:  exec2.ID,
	}
	_, err = q.CreateExecutionLink(ctx, exec1ToExec2Params)
	if err != nil {
		testLogger.Error("failed to create execution link", zap.Error(err))
		return err
	}

	exec31Params := entities.CreateExecutionParams{
		ID:   "exec31",
		Kind: "mention",
	}
	exec31, err := q.CreateExecution(ctx, exec31Params)
	if err != nil {
		testLogger.Error("failed to create execution", zap.Error(err))
		return err
	}

	exec31ToExec2Params := entities.CreateExecutionLinkParams{
		ParentID: exec2.ID,
		ChildID:  exec31.ID,
	}
	_, err = q.CreateExecutionLink(ctx, exec31ToExec2Params)
	if err != nil {
		testLogger.Error("failed to create execution link", zap.Error(err))
		return err
	}

	exec32Params := entities.CreateExecutionParams{
		ID:   "exec32",
		Kind: "mention",
	}
	exec32, err := q.CreateExecution(ctx, exec32Params)
	if err != nil {
		testLogger.Error("failed to create execution", zap.Error(err))
		return err
	}

	exec32ToExec2Params := entities.CreateExecutionLinkParams{
		ParentID: exec2.ID,
		ChildID:  exec32.ID,
	}
	_, err = q.CreateExecutionLink(ctx, exec32ToExec2Params)
	if err != nil {
		testLogger.Error("failed to create execution link", zap.Error(err))
		return err
	}

	return

}
