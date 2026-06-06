package OpenTeam

import (
	"github.com/TigranOhanyan/OpenTeam/entities"
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
