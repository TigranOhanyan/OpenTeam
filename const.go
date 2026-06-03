package OpenTeam

const (
	AmountOfChoices   int64   = 1
	Temperature       float64 = 1.0
	ParallelToolCalls         = false
)

type Visibility string

const (
	VisibilityChannel Visibility = "channel"
	VisibilityRole    Visibility = "role"
	VisibilityTask    Visibility = "task"
)

type EventKind string

const (
	EventKindAsking    EventKind = "asking"
	EventKindReasoning EventKind = "reasoning"
	EventKindActing    EventKind = "acting"
)
