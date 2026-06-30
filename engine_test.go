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
	assert.Equal(t, 2, len(executionReports.Reports))

	startOfCheck := time.Now()
	startOfCheck = startOfCheck.Add(time.Second)

	actualContractingExecutionReport := executionReports.Reports[0]

	assert.Equal(t, actualContractingExecutionReport.ExecutionID, "exec2")
	assert.Equal(t, actualContractingExecutionReport.Status, ExecutionStatusClosed)

	actualContractingExecution, err := teamDb.Queries.GetExecution(ctx, "exec2")
	assert.NoError(t, err)
	assert.Equal(t, actualContractingExecution.Status, "closed")

	actualSkippedExecutionReport := executionReports.Reports[1]
	assert.Equal(t, actualSkippedExecutionReport.ExecutionID, "exec1")
	assert.Equal(t, actualSkippedExecutionReport.Status, ExecutionStatusSkipped)
	actualSkippedExecution, err := teamDb.Queries.GetExecution(ctx, "exec1")
	assert.NoError(t, err)
	assert.Equal(t, actualSkippedExecution.Status, "open")

	assert.False(t, executionReports.AreAllClosed())
	assert.False(t, executionReports.IsIdle())

}

func Test_Engine_when_called_consecutively_should_resolve_the_grandparent_execution_if_all_children_and_grandchildren_are_resolved_and_resolver_is_contracting(t *testing.T) {
	var err error

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

	actualContractingExecutionReport := executionReports.Reports[0]

	assert.Equal(t, actualContractingExecutionReport.ExecutionID, "exec1")
	assert.Equal(t, actualContractingExecutionReport.Status, ExecutionStatusClosed)

	actualContractingExecution, err := teamDb.Queries.GetExecution(ctx, "exec1")
	assert.NoError(t, err)
	assert.Equal(t, actualContractingExecution.Status, "closed")

	assert.True(t, executionReports.AreAllClosed())
	assert.False(t, executionReports.IsIdle())

}

func Test_Engine_when_all_executions_are_resolved_should_return_an_empty_report(t *testing.T) {
	var err error

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

	assert.True(t, executionReports.AreAllClosed())
	assert.True(t, executionReports.IsIdle())
}

func Test_Engine_should_spawn_an_execution_if_all_children_are_resolved_and_resolver_is_expanding(t *testing.T) {
	var err error

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
	assert.Equal(t, 2, len(executionReports.Reports))

	startOfCheck := time.Now()
	startOfCheck = startOfCheck.Add(time.Second)

	actualExpandingExecutionReport := executionReports.Reports[0]

	assert.Equal(t, actualExpandingExecutionReport.ExecutionID, "exec2")
	assert.Equal(t, actualExpandingExecutionReport.Status, ExecutionStatusExpanded)

	actualExpandingExecution, err := teamDb.Queries.GetExecution(ctx, "exec2")
	assert.NoError(t, err)
	assert.Equal(t, actualExpandingExecution.Status, "open")

	actualNewExecution, err := teamDb.Queries.GetExecution(ctx, "exec2-next")
	assert.NoError(t, err)
	assert.Equal(t, actualNewExecution.Status, "open")
	assert.WithinRange(t, actualNewExecution.CreatedAt, startOfTest, startOfCheck)

	actualSkippedExecutionReport := executionReports.Reports[1]
	assert.Equal(t, actualSkippedExecutionReport.ExecutionID, "exec1")
	assert.Equal(t, actualSkippedExecutionReport.Status, ExecutionStatusSkipped)
	actualSkippedExecution, err := teamDb.Queries.GetExecution(ctx, "exec1")
	assert.NoError(t, err)
	assert.Equal(t, actualSkippedExecution.Status, "open")

	assert.False(t, executionReports.AreAllClosed())
	assert.False(t, executionReports.IsIdle())
}

func Test_Engine_when_called_consecutively_should_return_empty_report_if_resolver_is_expanding(t *testing.T) {
	var err error

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
	assert.Equal(t, 2, len(executionReports.Reports))

	actualFirstSkippedExecutionReport := executionReports.Reports[0]
	assert.Equal(t, actualFirstSkippedExecutionReport.ExecutionID, "exec2-next")
	assert.Equal(t, actualFirstSkippedExecutionReport.Status, ExecutionStatusSkipped)
	actualFirstSkippedExecution, err := teamDb.Queries.GetExecution(ctx, "exec2-next")
	assert.NoError(t, err)
	assert.Equal(t, actualFirstSkippedExecution.Status, "open")

	actualSecondSkippedExecutionReport := executionReports.Reports[1]
	assert.Equal(t, actualSecondSkippedExecutionReport.ExecutionID, "exec2")
	assert.Equal(t, actualSecondSkippedExecutionReport.Status, ExecutionStatusSkipped)
	actualSecondSkippedExecution, err := teamDb.Queries.GetExecution(ctx, "exec2")
	assert.NoError(t, err)
	assert.Equal(t, actualSecondSkippedExecution.Status, "open")

	actualThirdSkippedExecutionReport := executionReports.Reports[2]
	assert.Equal(t, actualThirdSkippedExecutionReport.ExecutionID, "exec1")
	assert.Equal(t, actualThirdSkippedExecutionReport.Status, ExecutionStatusSkipped)

	actualThirdSkippedExecution, err := teamDb.Queries.GetExecution(ctx, "exec1")
	assert.NoError(t, err)
	assert.Equal(t, actualThirdSkippedExecution.Status, "open")

	assert.False(t, executionReports.AreAllClosed())
	assert.True(t, executionReports.IsIdle())
}

type contractingResolver struct {
	conversationHistoryDb *TeamDb
}

func (r *contractingResolver) Resolve(
	ctx context.Context,
	execution entities.Execution,
	logger *zap.Logger,
) (
	executionReport ExecutionReport,
	err error,
) {

	executionReport.ExecutionID = execution.ID

	trx, err := r.conversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := r.conversationHistoryDb.Queries.WithTx(trx)

	children, err := qtx.GetChildExecutions(ctx, execution.ID)
	if err != nil {
		logger.Error("failed to get child executions", zap.Error(err))
		return
	}

	allChildrenClosed := true
	for _, child := range children {
		if child.Status != "closed" {
			allChildrenClosed = false
			break
		}
	}

	if !allChildrenClosed {
		executionReport.Status = ExecutionStatusSkipped
		return
	}

	_, err = qtx.CloseExecution(ctx, execution.ID)
	if err != nil {
		logger.Error("failed to close execution", zap.Error(err))
		return
	}
	executionReport.Status = ExecutionStatusClosed
	return

}

type expandingResolver struct {
	conversationHistoryDb *TeamDb
}

func (r *expandingResolver) Resolve(
	ctx context.Context,
	execution entities.Execution,
	logger *zap.Logger,
) (
	executionReport ExecutionReport,
	err error,
) {

	trx, err := r.conversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := r.conversationHistoryDb.Queries.WithTx(trx)

	executionReport.ExecutionID = execution.ID

	children, err := qtx.GetChildExecutions(ctx, execution.ID)
	if err != nil {
		logger.Error("failed to get child executions", zap.Error(err))
		return
	}

	if len(children) == 0 {
		executionReport.Status = ExecutionStatusExpanded
		return
	}

	allChildrenClosed := true
	for _, child := range children {
		if child.Status != "closed" {
			allChildrenClosed = false
			break
		}
	}

	if !allChildrenClosed {
		executionReport.Status = ExecutionStatusSkipped
		return

	}

	newExecId := fmt.Sprintf("%s-next", execution.ID)

	newExecutionParams := entities.CreateExecutionParams{
		ID:   newExecId,
		Kind: execution.Kind,
	}

	newExecution, er := qtx.CreateExecution(ctx, newExecutionParams)
	err = er
	if err != nil {
		logger.Error("failed to create execution", zap.Error(err))
		return
	}

	newExecutionLinkParams := entities.CreateExecutionLinkParams{
		ParentID: execution.ID,
		ChildID:  newExecution.ID,
	}
	_, err = qtx.CreateExecutionLink(ctx, newExecutionLinkParams)
	if err != nil {
		logger.Error("failed to create execution link", zap.Error(err))
		return
	}
	executionReport.Status = ExecutionStatusExpanded
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
