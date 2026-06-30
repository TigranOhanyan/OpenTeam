package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

type mentionResolver struct {
	team *Team
}

func (r *mentionResolver) Resolve(
	ctx context.Context,
	mentionExecution entities.Execution,
	logger *zap.Logger,
) (executionReport ExecutionReport, err error) {

	executionReport.ExecutionID = mentionExecution.ID

	logger = logger.With(zap.String("mentionExecutionId", mentionExecution.ID))
	logger.Info("resolving mention...")

	trx, err := r.team.ConversationHistoryDb.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.Error("failed to begin transaction", zap.Error(err))
		return
	}

	defer func() {
		if err != nil {
			trx.Rollback()
			return
		}
		err = trx.Commit()
		if err != nil {
			logger.Error("failed to commit transaction", zap.Error(err))
			return
		}
		return
	}()

	qtx := r.team.ConversationHistoryDb.Queries.WithTx(trx)

	allChildren, err := qtx.GetChildExecutions(ctx, mentionExecution.ID)
	if err != nil {
		logger.Error("failed to get child executions", zap.Error(err))
		return
	}

	hasChildren := len(allChildren) > 0

	if hasChildren {
		allChildrenClosed := true // todo implement the next task expansion
		for _, child := range allChildren {
			if child.Status != "closed" {
				allChildrenClosed = false
				break
			}
		}
		if !allChildrenClosed {
			executionReport.Status = ExecutionStatusSkipped
			return
		}
		executionReport.Status = ExecutionStatusClosed
		_, err = qtx.CloseExecution(ctx, mentionExecution.ID)
		if err != nil {
			logger.Error("failed to close execution", zap.Error(err))
			return
		}
		return
	}

	mentionRecord, err := qtx.GetMention(ctx, mentionExecution.ID)
	if err != nil {
		logger.Error("failed to get mention", zap.Error(err))
		return
	}

	fromRoleRecord, err := qtx.GetRole(ctx, mentionRecord.FromMemberRoleID)
	if err != nil {
		logger.Error("failed to get role", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("roleId", fromRoleRecord.ID))

	channelRecord, err := qtx.GetChannel(ctx, fromRoleRecord.ChannelName)
	if err != nil {
		logger.Error("failed to get channel", zap.Error(err))
		return
	}
	logger = logger.With(zap.String("channelId", channelRecord.Name))

	memberRecord, err := qtx.GetMember(ctx, mentionRecord.ToMemberName)
	if err != nil {
		logger.Error("failed to get to participant", zap.Error(err))
		return
	}

	roleByMemberAndChannelParams := entities.GetRoleByMemberAndChannelParams{
		MemberName:  memberRecord.Name,
		ChannelName: channelRecord.Name,
	}

	roleRecord, err := qtx.GetRoleByMemberAndChannel(ctx, roleByMemberAndChannelParams)
	if err != nil {
		logger.Error("failed to get to membership", zap.Error(err))
		return
	}

	taskRecord, err := qtx.GetFirstTask(ctx, roleRecord.ID)
	if err != nil {
		logger.Error("failed to get to persona", zap.Error(err))
		return
	}

	rawTaskExecution := rawTaskExecution{
		parentExecutionID: mentionExecution.ID,
		taskID:            taskRecord.ID,
	}

	_, err = rawTaskExecution.persist(ctx, qtx, logger)
	if err != nil {
		logger.Error("failed to persist task execution", zap.Error(err))
		return
	}

	executionReport.Status = ExecutionStatusExpanded
	return

}
