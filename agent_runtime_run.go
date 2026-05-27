package OpenTeam

import (
	"context"

	"go.uber.org/zap"
)

func (runtime *AgentRuntime) Run(
	ctx context.Context,
	askingStepId string,
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

	orchestrationPlan, err := reconstitutePlan(ctx, qtx, askingStepId, logger)
	if err != nil {
		logger.Error("failed to reconstitute plan", zap.Error(err))
		return
	}

	err = trx.Commit()
	if err != nil {
		logger.Error("failed to commit transaction", zap.Error(err))
		return
	}

	for _, mentionRecord := range orchestrationPlan.mentionRecords {

		agent, er := runtime.createAgent(ctx, mentionRecord, logger)
		err = er
		if err != nil {
			logger.Error("failed to create agent", zap.Error(err))
			return
		}

		reply, err = agent.reAct(ctx, logger)
		if err != nil {
			logger.Error("failed to reAct mention", zap.Error(err))
			return
		}
	}

	return

}
