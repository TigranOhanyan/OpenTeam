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

type CdcEvent struct {
	Kind        CdcEventKind
	ChannelName string
	MemberName  string
	Action      *entities.Action
	Chunk       *entities.LlmChunkResponse
	Mention     *entities.Mention
}

type EventKind string

const (
	EventKindCdc        EventKind = "cdc"
	EventKindLoopError  EventKind = "loop_error"
	EventKindLoopReport EventKind = "loop_report"
)

// type Event struct {
// 	Kind EventKind
// 	Data interface{}
// }

type Event struct {
	Kind     EventKind
	CdcEvent *CdcEvent
	Error    *error
	Report   *RunReport
}

type RunReport struct {
	ExecutedRunIds []string
	SkippedRunIds  []string
}

func (summary *RunReport) AllRunsCompleted() bool {
	return len(summary.ExecutedRunIds)+len(summary.SkippedRunIds) == 0
}

func (summary *RunReport) NoReadyRunToExecute() bool {
	return len(summary.ExecutedRunIds) == 0
}
