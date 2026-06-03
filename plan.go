package OpenTeam

import (
	"github.com/TigranOhanyan/OpenTeam/entities"
)

type Plan struct {
	mentionRecords []entities.Mention
	actingStepIds  []string
}

func (o *Plan) isOnlyControl() bool {
	return len(o.mentionRecords) != 0 && len(o.actingStepIds) == 0
}

func (o *Plan) isFinalReply() bool {
	return len(o.mentionRecords) == 0 && len(o.actingStepIds) == 0
}
