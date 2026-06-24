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
	streamRO <-chan Event,
) {

	// TODO implement some sort of safe error handling that will stop the for-select loop
	// TODO it woudl be better to extend the cdc events to capture the possible errors too
	// TODO the entire engine form the business perpective can only fail on the openAI call - this case should be thought-out
	// TODO possible solution is to notify the caller about the error and allow a possiblity of retries
	// TODO other type of operations are done on a local sqlite file so from the business perspective no error is expected
	// TDOD although any technical error (file corruption or programmer bug) can take place so a safe mechanisme should be thought out
	// TODO the simplest possible solution is to return the error and stop the execution of Run
	stream := make(chan Event)
	streamRO = stream

	go func() {
		defer close(stream)

		// Single-threaded synchronous event loop.
		// We continually pull incomplete runs and their children without any lead ID,
		// do the complex filtering in Go, and then advance the state.
		for {

			select {
			case <-ctx.Done():
				return

			default:

				runSummary, err := runtime.run_iteration(ctx, stream, logger)
				if err != nil {
					logger.Error("failed to run iteration", zap.Error(err))
					return
				}

				event := Event{
					Kind:   EventKindLoopReport,
					Report: &runSummary,
				}
				select {
				case <-ctx.Done():
					return
				case stream <- event:
				default:
				}

			}

		}
	}()

	return

}

func (runtime *AgentRuntime) run_iteration(
	ctx context.Context,
	stream chan<- Event,
	logger *zap.Logger,
) (
	runSummary RunReport,
	err error,
) {

	runSummary = RunReport{
		ExecutedRunIds: make([]string, 0),
		SkippedRunIds:  make([]string, 0),
	}

	time.Sleep(100 * time.Millisecond)

	incompleteRuns, err := runtime.ConversationHistoryDb.Queries.GetIncompleteRunsAndChildren(ctx)
	if err != nil {
		logger.Error("failed to get incomplete runs", zap.Error(err))
		return
	}

	if len(incompleteRuns) == 0 {
		return
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
			runSummary.SkippedRunIds = append(runSummary.SkippedRunIds, runID)
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
			runSummary.SkippedRunIds = append(runSummary.SkippedRunIds, runID)
			continue // Waiting on children
		}

		runSummary.ExecutedRunIds = append(runSummary.ExecutedRunIds, runID)

		agentReActLoop, err := runtime.createAgent(ctx, runID, logger)
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

		err = agentReActLoop.reAct(ctx, stream, logger)
		if err != nil {
			logger.Error("failed to re-act", zap.Error(err))
			break
		}
	}

	return

}
