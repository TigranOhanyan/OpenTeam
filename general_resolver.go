package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type generalResolver struct {
	team *Team
}

func (r *generalResolver) Resolve(
	ctx context.Context,
	execution entities.Execution,
	logger *zap.Logger,
) (executionReport ExecutionReport, err error) {
	executionReport.ExecutionID = execution.ID

	switch execution.Kind {
	case "task":
		taskExecutionResolver := taskExecutionResolver{team: r.team}
		executionReport, err = taskExecutionResolver.Resolve(ctx, execution, logger)
		return
	case "react":
		reactExecutionResolver := reactExecutionResolver{team: r.team}
		executionReport, err = reactExecutionResolver.Resolve(ctx, execution, logger)
		return
	case "ask":
		executionReport.Status = ExecutionStatusSkipped
		return
	case "tool":
		executionReport.Status = ExecutionStatusSkipped
		return
	case "reason":
		executionReport.Status = ExecutionStatusSkipped
		return
	case "agent":
		executionReport.Status = ExecutionStatusSkipped
		return
	case "act":
		actExecutionResolver := actExecutionResolver{team: r.team}
		executionReport, err = actExecutionResolver.Resolve(ctx, execution, logger)
		return
	case "mention":
		mentionResolver := mentionResolver{team: r.team}
		executionReport, err = mentionResolver.Resolve(ctx, execution, logger)
		return
	}

	return
}
