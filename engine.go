package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type ExecutionResolver interface {
	Resolve(
		ctx context.Context,
		qtx *entities.Queries,
		parent entities.Execution,
		children []entities.Execution,
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
	trx, err := engine.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
		}
	}()

	qtx := engine.ConversationHistoryDb.Queries.WithTx(trx)

	resolvableExecutionCandidates, err := qtx.GetResolvableExecutionCandidates(ctx)
	if err != nil {
		logger.Error("failed to get resolvable execution candidates", zap.Error(err))
		return
	}

	resolvableExecutions := make(map[string]*parentGroup)
	for _, row := range resolvableExecutionCandidates {
		group, ok := resolvableExecutions[row.ID]
		if !ok {
			group = &parentGroup{
				parent: row,
			}

			children, er := qtx.GetChildExecutions(ctx, row.ID)
			err = er
			if err != nil {
				logger.Error("failed to get child executions", zap.Error(err))
				return
			}
			group.children = children

			isResolvable := group.AllChildrenResolved()

			if isResolvable {
				resolvableExecutions[row.ID] = group
			}
		}
	}

	executionReports = ExecutionReports{
		Reports: make([]ExecutionReport, 0, len(resolvableExecutions)),
	}

	for _, group := range resolvableExecutions {
		report, er := resolver.Resolve(ctx, qtx, group.parent, group.children, logger)
		if er != nil {
			err = er
			logger.Error(
				"failed to resolve execution",
				zap.Error(err),
				zap.String("parentId", group.parent.ID),
			)
			return
		}
		executionReports.Reports = append(executionReports.Reports, report)
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	remainingOpenExecutions, err := engine.ConversationHistoryDb.Queries.GetOpenExecutions(ctx)
	if err != nil {
		logger.Error("failed to get remaining open executions", zap.Error(err))
		return
	}
	executionReports.OpenExecutionIds = make([]string, 0, len(remainingOpenExecutions))
	for _, execution := range remainingOpenExecutions {
		executionReports.OpenExecutionIds = append(executionReports.OpenExecutionIds, execution.ID)
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

type ExecutionReport struct {
	Parent              entities.Execution
	NewlyOpenedChildren []entities.Execution
}

type ExecutionReports struct {
	Reports          []ExecutionReport
	OpenExecutionIds []string
}

func (reports *ExecutionReports) IsResolved() bool {
	return len(reports.OpenExecutionIds) == 0
}

func (reports *ExecutionReports) IsIdle() bool {
	return len(reports.Reports) == 0
}
