package openteam

const (
	AmountOfChoices   int64   = 1
	Temperature       float64 = 1.0
	ParallelToolCalls         = false
)

type Visibility string

const (
	VisibilityChannel Visibility = "channel"
	VisibilityRole    Visibility = "role"
	VisibilityDuty    Visibility = "duty"
)

type EventKind string

const (
	EventKindAddressing EventKind = "addressing"
	EventKindReplying   EventKind = "replying"
	EventKindPlanning   EventKind = "planning"
	EventKindThinking   EventKind = "thinking"
	EventKindActing     EventKind = "acting"
	EventKindActed      EventKind = "acted"
)

type TurnStatus string

const (
	TurnStatusPending   TurnStatus = "pending"
	TurnStatusCompleted TurnStatus = "completed"
)
