package openteam

type OrchestrationPlan struct {
	replyTurnId         *string
	articulationTurnIds []string
	actingTurnIds       []string
	handoffTurnIds      []string
}
