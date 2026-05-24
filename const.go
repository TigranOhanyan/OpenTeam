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
	VisibilityTask    Visibility = "task"
)

type EventKind string

const (
	EventKindMention   EventKind = "mention"
	EventKindObserving EventKind = "observing"
	EventKindReasoning EventKind = "reasoning"
	EventKindActing    EventKind = "acting"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusCompleted StepStatus = "completed"
)
