package OpenTeam

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TigranOhanyan/OpenTeam/entities"
	"go.uber.org/zap"
)

// sherlockHolmesAgency builds the "A Scandal in Bohemia" team ontology:
//
//	Members:  King (human), Holmes (bot, bridge), Watson (bot, has a tool)
//	Channels: #consulting-room (King + Holmes), #case-file (Holmes + Watson)
//
// Holmes bridges the two channels through a single chained task pipeline that
// walks #consulting-room -> #case-file -> #consulting-room. Watson owns the
// lookup_records tool, which yields to the host during the investigation.
const holmesInstrucionForReceive = "Receive the client's problem in the consulting room. If the client's case requires investigation, acknowledge it and move to the private investigation room. Otherwise, deliver a clean deduction to the client."

const holmesInstrucionForInvestigation = "In the case file, identify the person involved by asking Watson, then reason about the case."

const holmesInstrucionForReport = "Tell your thoughts to Watson about his findings and return to the consulting room and deliver a clean deduction to the client."

const watsonInstrucionForLookup = "Consult the index with the lookup_records tool, then give Holmes your observation."

const watsonInstrucionForLookupResult = "Irene Adler. Born New Jersey, 1858. Contralto. Prima donna, Imperial Opera of Warsaw; retired from the operatic stage. Resides at Briony Lodge, Serpentine Avenue, St. John's Wood, London."

const lookupRecordsTool = `{
    		        "type": "function",
    		        "function": {
    		            "name": "lookup_records",
    		            "description": "Look up a person in the index of records.",
    		            "parameters": {
    		                "type": "object",
    		                "properties": {
    		                    "name": {
    		                        "type": "string",
    		                        "description": "The full name of the person to look up."
    		                    }
    		                },
    		                "required": [
    		                    "name"
    		                ]
    		            }
    		        }
    		    }`

const (
	consultingRoom            = "consulting_room"
	consultingRoomDescription = "Client-facing consulting room at 221B Baker Street."
)

const (
	privateInvestigationRoom            = "private_investigation_room"
	privateInvestigationRoomDescription = "Private investigation room shared by Holmes and Watson."
)

func sherlockHolmesAgency(ctx context.Context, teamDb *TeamDb, logger *zap.Logger) (err error) {
	q := teamDb.Queries

	king, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "King",
		Kind: "human",
	})
	if err != nil {
		logger.Error("failed to create member", zap.Error(err))
		return err
	}

	holmes, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "Holmes",
		Kind: "bot",
	})
	if err != nil {
		logger.Error("failed to create member", zap.Error(err))
		return err
	}

	watson, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "Watson",
		Kind: "bot",
	})
	if err != nil {
		logger.Error("failed to create member", zap.Error(err))
		return err
	}

	consultingRoom, err := q.CreateChannel(ctx, entities.CreateChannelParams{
		Name:        consultingRoom,
		Description: consultingRoomDescription,
	})
	if err != nil {
		logger.Error("failed to create channel", zap.Error(err))
		return err
	}

	caseFile, err := q.CreateChannel(ctx, entities.CreateChannelParams{
		Name:        privateInvestigationRoom,
		Description: privateInvestigationRoomDescription,
	})
	if err != nil {
		logger.Error("failed to create channel", zap.Error(err))
		return err
	}

	// King: the client, present only in the consulting room.
	_, err = q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "king-at-consulting-room",
		MemberName:  king.Name,
		ChannelName: consultingRoom.Name,
	})
	if err != nil {
		logger.Error("failed to create role", zap.Error(err))
		return err
	}

	// Holmes: the bridge. He holds a role in both channels and a single task
	// pipeline that crosses from the consulting room into the case file and back.
	holmesConsultingRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "holmes-at-consulting-room",
		MemberName:  holmes.Name,
		ChannelName: consultingRoom.Name,
	})
	if err != nil {
		logger.Error("failed to create role", zap.Error(err))
		return err
	}

	holmesCaseFileRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "holmes-at-case-file",
		MemberName:  holmes.Name,
		ChannelName: caseFile.Name,
	})
	if err != nil {
		logger.Error("failed to create role", zap.Error(err))
		return err
	}

	holmesReceive, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "holmes-consulting-receive",
		RoleID:      holmesConsultingRole.ID,
		Instruction: holmesInstrucionForReceive,
	})
	if err != nil {
		logger.Error("failed to create task", zap.Error(err))
		return err
	}

	fmt.Println("holmesReceive", holmesReceive)

	holmesInvestigate, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "holmes-case-file-investigate",
		RoleID:      holmesCaseFileRole.ID,
		Instruction: holmesInstrucionForInvestigation,
	})
	if err != nil {
		logger.Error("failed to create task", zap.Error(err))
		return err
	}

	holmesReport, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "holmes-consulting-report",
		RoleID:      holmesConsultingRole.ID,
		Instruction: holmesInstrucionForReport,
	})
	if err != nil {
		logger.Error("failed to create task", zap.Error(err))
		return err
	}

	_, err = q.CreateTaskLink(ctx, entities.CreateTaskLinkParams{
		ParentID: holmesInvestigate.ID,
		ChildID:  holmesReport.ID,
	})
	if err != nil {
		logger.Error("failed to create task link", zap.Error(err))
		return err
	}

	// Watson: the specialist in the case file, equipped with the records tool.
	watsonCaseFileRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "watson-at-case-file",
		MemberName:  watson.Name,
		ChannelName: caseFile.Name,
	})
	if err != nil {
		logger.Error("failed to create role", zap.Error(err))
		return err
	}

	watsonLookup, err := q.CreateTask(ctx, entities.CreateTaskParams{
		ID:          "watson-case-file-lookup",
		RoleID:      watsonCaseFileRole.ID,
		Instruction: watsonInstrucionForLookup,
	})
	if err != nil {
		logger.Error("failed to create task", zap.Error(err))
		return err
	}

	_, err = q.CreateTool(ctx, entities.CreateToolParams{
		ID:     "watson-case-file-lookup-records-tool",
		TaskID: watsonLookup.ID,
		Tool:   json.RawMessage(lookupRecordsTool),
	})
	if err != nil {
		logger.Error("failed to create tool", zap.Error(err))
		return err
	}

	return nil
}

const kingsInquery = `Mr. Holmes, I am in a most delicate difficulty. There exists a photograph — a cabinet portrait — of myself and a certain lady. She threatens to send it to the household of my betrothed on the day my engagement is announced. I must have it recovered. Discretion is everything.`

const holmesInitialResponse = `Pray be precise. A photograph compromises only if its provenance is certain — so the lady is a person of resolution, and no common adventuress. Give me the name, and a day to make my inquiries, and you shall have your answer. Say nothing further to anyone.`

const holmesRequestToWatson = `Watson, the King of Bohemia brings us a photograph and a woman. Before we move a step, let us know precisely whom we are dealing with. The name is Irene Adler. Hand me the index.`

const watsonFileLookupResult = `Irene Adler. Born New Jersey, 1858. Contralto. Prima donna, Imperial Opera of Warsaw; retired from the operatic stage. Resides at Briony Lodge, Serpentine Avenue, St. John's Wood, London.`

const watsonResponseToHolmes = `A retired prima donna living quietly, then. Hardly a blackmailer by trade. If she keeps the photograph, Holmes, she keeps it as a weapon of defence — not for sale.`

const holmesResponseToWatson = `Precisely my own conclusion. She will not surrender it to threats, nor to money — only to the certainty that producing it can no longer serve her. The photograph is not in a bank; a woman of her spirit keeps such a thing within reach, at home. We do not need the law, Watson. We need to know where in that house she would hide it — and for that, she must show us herself.`

const holmesResponseToKing = `Your Majesty, the lady is no mercenary; she holds the photograph only to protect herself, and keeps it close at hand in her own house. Threats and gold will fail. Within three days I shall arrange matters so that she reveals its hiding place of her own accord, and the portrait will be in your hands. You need do nothing but wait.`

var _1_llm_request = fmt.Sprintf(`
{
	"model": "gpt-5",
	"messages": [
		{"role": "system", "content": "%s"},
		{"role": "user", "content": "%s", "name": "King"}
	],
	"tools": [
    		    {
    		        "type": "function",
    		        "function": {
    		            "name": "go_to_private_investigation_room",
    		            "description": "Private investigation room shared by Holmes and Watson.",
    		        }
    		    }
    		]
}
`, holmesInstrucionForInvestigation, kingsInquery)
