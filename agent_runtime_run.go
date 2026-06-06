package OpenTeam

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type runAndChildrenStatus struct {
	runId    string
	runKind  string
	children map[string]string
}

func (runtime *AgentRuntime) Run(
	ctx context.Context,
	logger *zap.Logger,
) (
	executedRuns ExecutedRuns,
	err error,
) {
	defer func() {
		if runtime.ChangeStream != nil {
			close(runtime.ChangeStream)
		}
	}()

	// Single-threaded synchronous event loop.
	// We continually pull incomplete runs and their children without any lead ID,
	// do the complex filtering in Go, and then advance the state.
	for {

		executedRunsOfIteration := make([]string, 0)

		time.Sleep(100 * time.Millisecond)

		incompleteRuns, err := runtime.ConversationHistoryDb.Queries.GetIncompleteRunsAndChildren(ctx)
		if err != nil {
			logger.Error("failed to get incomplete runs", zap.Error(err))
		}

		if len(incompleteRuns) == 0 {
			break
		}

		// Group by parent run
		childrenStatusByRun := make(map[string]runAndChildrenStatus)
		for _, row := range incompleteRuns {
			if _, exists := childrenStatusByRun[row.RunID]; !exists {
				childrenStatusByRun[row.RunID] = runAndChildrenStatus{
					runId:    row.RunID,
					runKind:  row.RunKind,
					children: make(map[string]string),
				}
			}
			if row.ChildRunID.Valid && row.ChildStatus.Valid {
				childrenStatusByRun[row.RunID].children[row.ChildRunID.String] = row.ChildStatus.String
			}
		}

		for runID, children := range childrenStatusByRun {
			// Check if all children are completed

			if children.runKind != "mention" {
				continue
			}

			allChildrenCompleted := true
			for _, status := range children.children {
				if status != "completed" {
					allChildrenCompleted = false
					break
				}
			}

			if !allChildrenCompleted {
				continue // Waiting on children
			}

			executedRunsOfIteration = append(executedRunsOfIteration, runID)

			agentReActLoop, er := runtime.createAgent(ctx, runID, logger)
			err = er
			if err != nil {
				logger.Error("failed to create agent", zap.Error(err))
				break
			}

			if agentReActLoop == nil {
				_, err = runtime.ConversationHistoryDb.Queries.CompleteRun(ctx, runID)
				if err != nil {
					logger.Error("failed to complete run", zap.Error(err))
					break
				}
				continue
			}

			err = agentReActLoop.reAct(ctx, logger)
			if err != nil {
				logger.Error("failed to re-act", zap.Error(err))
				break
			}
		}

		if len(executedRunsOfIteration) == 0 {
			break
		}

		executedRuns.RunIds = executedRunsOfIteration

	}

	return

}
