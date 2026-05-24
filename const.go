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
	EventKindMention  EventKind = "mention"
	EventKindReplying EventKind = "replying"
	EventKindPlanning EventKind = "planning"
	EventKindThinking EventKind = "thinking"
	EventKindActing   EventKind = "acting"
	EventKindActed    EventKind = "acted"
)

type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusCompleted StepStatus = "completed"
)
