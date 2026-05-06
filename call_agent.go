package openteam

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

func (a *Agent) callAgent(
	ctx context.Context,
	previousTurnId string,
	eventKind EventKind,
	logger *zap.Logger,
) (
	nextTurnId string,
	err error,
) {
	switch eventKind {
	case EventKindThinking:
		nextTurnId, err = a.callLLM(ctx, previousTurnId, logger)
		if err != nil {
			return
		}
		nextTurnId, err = a.callAgent(ctx, nextTurnId, EventKindThought, logger)
		return
	case EventKindThought:
		orchestrationPlan, er := a.dispatchThought(ctx, previousTurnId, logger)
		err = er
		if err != nil {
			return
		}
		if orchestrationPlan.replyTurnId != nil {
			nextTurnId = *orchestrationPlan.replyTurnId
			nextTurnId, err = a.callAgent(ctx, nextTurnId, EventKindReply, logger)
			if err != nil {
				return
			}
			return
		}

		if len(orchestrationPlan.articulationTurnIds) > 0 { // temp solution for now
			nextTurnId = orchestrationPlan.articulationTurnIds[0]
			nextTurnId, err = a.callAgent(ctx, nextTurnId, EventKindArticulation, logger)
			if err != nil {
				return
			}
			nextTurnId, err = a.callAgent(ctx, nextTurnId, EventKindThinking, logger)
			return
		}
		return
	case EventKindActing:
		err = fmt.Errorf("AgentHandoffRequested is not implemented yet")
		return
	case EventKindActed:
		err = fmt.Errorf("AgentHandoffRequested is not implemented yet")
		return
	case EventKindArticulation:
		nextTurnId, err = a.articulateToAgent(ctx, previousTurnId, logger)
		if err != nil {
			return
		}
		nextTurnId, err = a.callAgent(ctx, nextTurnId, EventKindThinking, logger)
		return
	case EventKindReply:
		nextTurnId = previousTurnId
		// todo: setting the name and changing the message role can be done here
		return
	default:
		return "", fmt.Errorf("unknown event kind: %s", eventKind)
	}
}
