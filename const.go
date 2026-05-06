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
	VisibilityHidden  Visibility = "hidden"
)

type EventKind string

const (
	EventKindArticulation EventKind = "articulation"
	EventKindReply        EventKind = "reply"
	EventKindThinking     EventKind = "thinking"
	EventKindThought      EventKind = "thought"
	EventKindActing       EventKind = "acting"
	EventKindActed        EventKind = "acted"
)

type TurnStatus string

const (
	TurnStatusPending   TurnStatus = "pending"
	TurnStatusCompleted TurnStatus = "completed"
)
