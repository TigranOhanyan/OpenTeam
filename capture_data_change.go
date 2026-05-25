package openteam

import (
	"context"

	"github.com/openteam/entities"
	"go.uber.org/zap"
)

type CdcEventKind string

const (
	CdcEventKindAction       CdcEventKind = "action"
	CdcEventKindMessage      CdcEventKind = "message"
	CdcEventKindMessageChunk CdcEventKind = "message_chunk"
	CdcEventKindMention      CdcEventKind = "mention"
)

type ChangeEvent struct {
	Kind        CdcEventKind
	RunID       string
	StepID      string
	ChannelName string
	MemberName  string
	TaskID      string
	Action      *entities.Action
	Chunk       *entities.LlmChunkResponse
	Mention     *entities.Mention
}

func (agent *agent) insertChunk(
	ctx context.Context,
	qtx *entities.Queries,
	params entities.CreateLlmChunkResponsesParams,
	logger *zap.Logger,
) error {
	_, err := qtx.CreateLlmChunkResponses(ctx, params)
	if err != nil {
		return err
	}

	if agent.runtime.ChangeStream != nil {
		roleRecord, err := qtx.GetRoleByTask(ctx, params.TaskID)
		if err != nil {
			return err
		}
		channelRecord, err := qtx.GetChannelByRole(ctx, roleRecord.ID)
		if err != nil {
			return err
		}

		agent.runtime.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindMessageChunk,
			RunID:       agent.runRecord.ID,
			StepID:      params.StepID,
			TaskID:      agent.task.ID,
			MemberName:  agent.member.Name,
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
