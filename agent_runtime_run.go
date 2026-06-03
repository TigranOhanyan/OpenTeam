package OpenTeam

import (
	"context"
	"time"

	"go.uber.org/zap"
)

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
		time.Sleep(100 * time.Millisecond)

		incompleteRuns, err := runtime.ConversationHistoryDb.Queries.GetIncompleteRunsAndChildren(ctx)
		if err != nil {
			logger.Error("failed to get incomplete runs", zap.Error(err))
		}

		if len(incompleteRuns) == 0 {
			break
		}

		// Group by parent run
		childrenStatusByRun := make(map[string]map[string]string)
		for _, row := range incompleteRuns {
			if _, exists := childrenStatusByRun[row.RunID]; !exists {
				childrenStatusByRun[row.RunID] = make(map[string]string)
			}
			if row.ChildRunID.Valid && row.ChildStatus.Valid {
				childrenStatusByRun[row.RunID][row.ChildRunID.String] = row.ChildStatus.String
			}
		}

		var agentReActLoop *agenticReActLoop

		for runID, children := range childrenStatusByRun {
			// Check if all children are completed
			allChildrenCompleted := true
			for _, status := range children {
				if status != "completed" {
					allChildrenCompleted = false
					break
				}
			}

			if !allChildrenCompleted {
				continue // Waiting on children
			}

			executedRuns.RunIds = append(executedRuns.RunIds, runID)

			maybeAgentReActLoop, err := runtime.createAgent(ctx, runID, logger)
			if err != nil {
				logger.Error("failed to create agent", zap.Error(err))
			}

			if maybeAgentReActLoop == nil {
				_, err = runtime.ConversationHistoryDb.Queries.CompleteRun(ctx, runID)
				if err != nil {
					logger.Error("failed to complete run", zap.Error(err))
				}
				continue
			}

			agentReActLoop = maybeAgentReActLoop
			break

		}

		if agentReActLoop != nil {
			err = agentReActLoop.reAct(ctx, logger)
			if err != nil {
				logger.Error("failed to re-act", zap.Error(err))
			}
		}

	}

	return

}
