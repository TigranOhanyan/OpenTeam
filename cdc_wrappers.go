package openteam

import (
	"context"

	"github.com/openteam/entities"
)

func (a *Agent) insertChunk(ctx context.Context, params entities.CreateLlmChunkResponsesParams) error {
	_, err := a.ConversationHistoryDb.Queries.CreateLlmChunkResponses(ctx, params)
	if err != nil {
		return err
	}

	if a.ChangeStream != nil {
		roleRecord, err := a.ConversationHistoryDb.Queries.GetRoleByDuty(ctx, params.DutyID)
		if err != nil {
			return err
		}
		channelRecord, err := a.ConversationHistoryDb.Queries.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		a.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindChunk,
			TurnID:      params.TurnID,
			ChannelName: channelRecord.Name,
			Chunk: &entities.LlmChunkResponse{
				ID:                  params.ID,
				SequenceNumber:      params.SequenceNumber,
				TurnID:              params.TurnID,
				DutyID:              params.DutyID,
				OpenaiChunkResponse: params.OpenaiChunkResponse,
			},
		}
	}
	return nil
}

func (a *Agent) insertAction(ctx context.Context, params entities.CreateActionParams) error {
	_, err := a.ConversationHistoryDb.Queries.CreateAction(ctx, params)
	if err != nil {
		return err
	}

	if a.ChangeStream != nil {
		dutyRecord, err := a.ConversationHistoryDb.Queries.GetDutyByTurn(ctx, params.TurnID)
		if err != nil {
			return err
		}
		roleRecord, err := a.ConversationHistoryDb.Queries.GetRoleByDuty(ctx, dutyRecord.ID)
		if err != nil {
			return err
		}
		channelRecord, err := a.ConversationHistoryDb.Queries.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		a.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindAction,
			TurnID:      params.TurnID,
			ChannelName: channelRecord.Name,
			Action: &entities.Action{
				ID:         params.ID,
				TurnID:     params.TurnID,
				ToolCallID: params.ToolCallID,
				Name:       params.Name,
				Arguments:  params.Arguments,
			},
		}
	}
	return nil
}

func (a *Agent) insertArticulation(ctx context.Context, params entities.CreateArticulationParams) error {
	_, err := a.ConversationHistoryDb.Queries.CreateArticulation(ctx, params)
	if err != nil {
		return err
	}

	if a.ChangeStream != nil {
		dutyRecord, err := a.ConversationHistoryDb.Queries.GetDutyByTurn(ctx, params.TurnID)
		if err != nil {
			return err
		}
		roleRecord, err := a.ConversationHistoryDb.Queries.GetRoleByDuty(ctx, dutyRecord.ID)
		if err != nil {
			return err
		}
		channelRecord, err := a.ConversationHistoryDb.Queries.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		a.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindArticulation,
			TurnID:      params.TurnID,
			ChannelName: channelRecord.Name,
			Articulation: &entities.Articulation{
				ID:             params.ID,
				TurnID:         params.TurnID,
				FromMemberName: params.FromMemberName,
				ToMemberName:   params.ToMemberName,
				ToolCallID:     params.ToolCallID,
				Message:        params.Message,
			},
		}
	}
	return nil
}
