package OpenTeam

import (
	"context"

	"github.com/TigranOhanyan/OpenTeam/entities"
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
	ChannelName string
	MemberName  string
	Action      *entities.Action
	Chunk       *entities.LlmChunkResponse
	Mention     *entities.Mention
}

func (agent *agenticReActLoop) insertChunk(
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

		agent.runtime.ChangeStream <- ChangeEvent{
			Kind:        CdcEventKindMessageChunk,
			MemberName:  agent.member.Name,
			ChannelName: agent.channel.Name,
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
