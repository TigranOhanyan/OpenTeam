package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type ExecutionResolver interface {
	Resolve(
		ctx context.Context,
		execution entities.Execution,
		logger *zap.Logger,
	) (executionReport ExecutionReport, err error)
}

type Engine struct {
	ConversationHistoryDb *TeamDb
}

func (engine *Engine) Advance(
	ctx context.Context,
	logger *zap.Logger,
	resolver ExecutionResolver,
) (
	executionReports ExecutionReports,
	err error,
) {
	openExecutions, err := engine.ConversationHistoryDb.Queries.GetOpenExecutions(ctx)
	if err != nil {
		logger.Error("failed to get resolvable execution candidates", zap.Error(err))
		return
	}

	executionReports = ExecutionReports{
		Reports: make([]ExecutionReport, 0, len(openExecutions)),
	}

	for _, execution := range openExecutions {
		report, er := resolver.Resolve(ctx, execution, logger)
		if er != nil {
			err = er
			logger.Error(
				"failed to resolve execution",
				zap.Error(err),
				zap.String("executionId", execution.ID),
			)
			return
		}
		executionReports.Reports = append(executionReports.Reports, report)
	}

	return

}

type parentGroup struct {
	parent   entities.Execution
	children []entities.Execution
}

func (group *parentGroup) AllChildrenResolved() bool {
	if len(group.children) == 0 {
		return false
	}
	for _, child := range group.children {
		if child.Status != "closed" {
			return false
		}
	}
	return true
}

type ExecutionStatus string

const (
	ExecutionStatusSkipped  ExecutionStatus = "skipped"
	ExecutionStatusClosed   ExecutionStatus = "closed"
	ExecutionStatusExpanded ExecutionStatus = "expanded"
)

type ExecutionReport struct {
	ExecutionID string
	Status      ExecutionStatus
}

type ExecutionReports struct {
	Reports []ExecutionReport
}

func (reports *ExecutionReports) AreAllClosed() bool {

	for _, report := range reports.Reports {
		if report.Status != ExecutionStatusClosed {
			return false
		}
	}
	return true
}

func (reports *ExecutionReports) IsIdle() bool {
	if len(reports.Reports) == 0 {
		return true
	}

	for _, report := range reports.Reports {
		if report.Status != ExecutionStatusSkipped {
			return false
		}
	}
	return true
}
