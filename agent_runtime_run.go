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
	reply Reply,
	err error,
) {
	defer func() {
		if runtime.ChangeStream != nil {
			close(runtime.ChangeStream)
		}
	}()

	trx, err := runtime.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}
	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := runtime.ConversationHistoryDb.Queries.WithTx(trx)

	// Single-threaded synchronous event loop.
	// We continually pull incomplete runs and their children without any lead ID,
	// do the complex filtering in Go, and then advance the state.
	for {
		time.Sleep(100 * time.Millisecond)

		incompleteRuns, err := qtx.GetIncompleteRunsAndChildren(ctx)
		if err != nil {
			logger.Error("failed to get incomplete runs", zap.Error(err))
			return reply, err
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

			maybeAgentReActLoop, err := runtime.createAgent(ctx, runID, logger)
			if err != nil {
				logger.Error("failed to create agent", zap.Error(err))
				return reply, err
			}

			if maybeAgentReActLoop == nil {
				_, err = qtx.CompleteRun(ctx, runID)
				if err != nil {
					logger.Error("failed to complete run", zap.Error(err))
					return reply, err
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
				return reply, err
			}
		}

	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	return

}
