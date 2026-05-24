package openteam

import (
	"context"

	"github.com/openteam/entities"
	"go.uber.org/zap"
)

func (a *Agent) insertChunk(
	ctx context.Context,
	qtx *entities.Queries,
	params entities.CreateLlmChunkResponsesParams,
	logger *zap.Logger,
) error {
	_, err := qtx.CreateLlmChunkResponses(ctx, params)
	if err != nil {
		return err
	}

	if a.ChangeStream != nil {
		roleRecord, err := qtx.GetRoleByTask(ctx, params.TaskID)
		if err != nil {
			return err
		}
		channelRecord, err := qtx.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		a.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindChunk,
			StepID:      params.StepID,
			ChannelName: channelRecord.Name,
			Chunk: &entities.LlmChunkResponse{
				ID:                  params.ID,
				SequenceNumber:      params.SequenceNumber,
				StepID:              params.StepID,
				TaskID:              params.TaskID,
				OpenaiChunkResponse: params.OpenaiChunkResponse,
			},
		}
	}
	return nil
}

func (a *Agent) insertAction(
	ctx context.Context,
	qtx *entities.Queries,
	params entities.CreateActionParams,
	logger *zap.Logger,
) error {
	_, err := qtx.CreateAction(ctx, params)
	if err != nil {
		return err
	}

	if a.ChangeStream != nil {
		taskRecord, err := qtx.GetTaskByStep(ctx, params.StepID)
		if err != nil {
			return err
		}
		roleRecord, err := qtx.GetRoleByTask(ctx, taskRecord.ID)
		if err != nil {
			return err
		}
		channelRecord, err := qtx.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		a.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindAction,
			StepID:      params.StepID,
			ChannelName: channelRecord.Name,
			Action: &entities.Action{
				ID:         params.ID,
				StepID:     params.StepID,
				ToolCallID: params.ToolCallID,
				Name:       params.Name,
				Arguments:  params.Arguments,
			},
		}
	}
	return nil
}

func (a *Agent) insertMention(
	ctx context.Context,
	qtx *entities.Queries,
	params entities.CreateMentionParams,
	logger *zap.Logger,
) error {
	_, err := qtx.CreateMention(ctx, params)
	if err != nil {
		return err
	}

	if a.ChangeStream != nil {
		taskRecord, err := qtx.GetTaskByStep(ctx, params.StepID)
		if err != nil {
			return err
		}
		roleRecord, err := qtx.GetRoleByTask(ctx, taskRecord.ID)
		if err != nil {
			return err
		}
		channelRecord, err := qtx.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		a.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindMention,
			StepID:      params.StepID,
			ChannelName: channelRecord.Name,
			Mention: &entities.Mention{
				ID:               params.ID,
				StepID:           params.StepID,
				FromMemberTaskID: params.FromMemberTaskID,
				ToMemberName:     params.ToMemberName,
				ToolCallID:       params.ToolCallID,
				Message:          params.Message,
			},
		}
	}
	return nil
}
