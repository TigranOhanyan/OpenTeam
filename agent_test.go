package openteam

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"testing"
	"time"

	"github.com/openteam/entities"
	"github.com/stretchr/testify/assert"
)

func Test_Agent_should_acknowledge_the_user_message(t *testing.T) {
	var err error
	defer wiremockClient.Reset()
	agent := agentProto

	startOfTest := time.Now()
	startOfTest = startOfTest.Add(-time.Second)

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	teamDb, err := teamDbFactory.NewTeamDb(ctx, "aganet_should_acknowledge_the_user_message.db", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	err = makeTeamForAcknowledgeTest(ctx, teamDb)
	assert.NoError(t, err)

	agent.ConversationHistoryDb = teamDb

	messageId, err := agent.Acknowledge(ctx, "Jim", "lobby", "Hello Jane!", testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, messageId)

	actualUserTurns, err := teamDb.Queries.GetTurns(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, actualUserTurns)
	actualUserThoughtTurn := actualUserTurns[0]
	assert.NoError(t, err)
	assert.Equal(t, actualUserThoughtTurn.Kind, string(EventKindThought))
	assert.Equal(t, actualUserThoughtTurn.Status, string(TurnStatusCompleted))
	actualUserThoughtMessage, err := teamDb.Queries.GetMessageByTurn(ctx, actualUserThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualUserThoughtMessage.Visibility, string(VisibilityHidden))
	actualUserThoughtMessageJson, err := json.Marshal(actualUserThoughtMessage.OpenaiMessage)
	assert.NoError(t, err)
	expectedUserThoughtMessageJson := fmt.Sprintf(`{"name":"Jim","tool_calls":[{"id":"%s","function":{"arguments":"{\"agent_name\":\"Jane\",\"message\":\"Hello Jane!\"}","name":"articulate_to_agent"},"type":"function"}],"role":"assistant"}`, messageId)
	assert.JSONEq(t, expectedUserThoughtMessageJson, string(actualUserThoughtMessageJson))

	actualArticulation, err := teamDb.Queries.GetArticulationByTurn(ctx, actualUserThoughtTurn.ID)
	assert.NoError(t, err)
	assert.Equal(t, actualArticulation.ToolCallID, messageId)
	assert.Equal(t, actualArticulation.Message, "Hello Jane!")

	articulationFromMember, err := teamDb.Queries.GetMember(ctx, actualArticulation.FromMemberName)
	assert.NoError(t, err)
	assert.Equal(t, articulationFromMember.Name, "Jim")
	articulationToMember, err := teamDb.Queries.GetMember(ctx, actualArticulation.ToMemberName)
	assert.NoError(t, err)
	assert.Equal(t, articulationToMember.Name, "Jane")
}

func makeTeamForAcknowledgeTest(ctx context.Context, teamDb *TeamDb) (err error) {

	q := teamDb.Queries

	jane, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "Jane",
		Kind: "bot",
	})
	if err != nil {
		testLogger.Error("failed to create member", zap.Error(err))
		return err
	}

	john, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "John",
		Kind: "bot",
	})
	if err != nil {
		testLogger.Error("failed to create member", zap.Error(err))
		return err
	}

	jim, err := q.CreateMember(ctx, entities.CreateMemberParams{
		Name: "Jim",
		Kind: "human",
	})
	if err != nil {
		testLogger.Error("failed to create member", zap.Error(err))
		return err
	}

	lobby, err := q.CreateChannel(ctx, entities.CreateChannelParams{
		Name:        "lobby",
		Description: "Lobby channel",
	})
	if err != nil {
		testLogger.Error("failed to create channel", zap.Error(err))
		return err
	}

	warRoom, err := q.CreateChannel(ctx, entities.CreateChannelParams{
		Name:        "war-room",
		Description: "War room channel",
	})
	if err != nil {
		testLogger.Error("failed to create channel", zap.Error(err))
		return err
	}

	janeLobbyRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "jane-at-lobby",
		MemberName:  jane.Name,
		ChannelName: lobby.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	janeWarRoomRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "jane-at-war-room",
		MemberName:  jane.Name,
		ChannelName: warRoom.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	janeLobbyFirstImpression, err := q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jane-lobby-first-impression",
		RoleID:      janeLobbyRole.ID,
		Instruction: "You are a first impression in the lobby.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	janeLobbyDecisionMake, err := q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jane-lobby-decision-make",
		RoleID:      janeLobbyRole.ID,
		PrevID:      sql.NullString{String: janeLobbyFirstImpression.ID, Valid: true},
		Instruction: "You are a decision maker in the lobby.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	_, err = q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jane-war-room-coordinator",
		RoleID:      janeWarRoomRole.ID,
		PrevID:      sql.NullString{String: janeLobbyDecisionMake.ID, Valid: true},
		Instruction: "You are a coordinator in the war room.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	johnWarRoomRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "john-at-war-room",
		MemberName:  john.Name,
		ChannelName: warRoom.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	_, err = q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "john-war-room-expert",
		RoleID:      johnWarRoomRole.ID,
		Instruction: "You are a expert in the war room.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	jimLobbyRole, err := q.CreateRole(ctx, entities.CreateRoleParams{
		ID:          "jim-at-lobby",
		MemberName:  jim.Name,
		ChannelName: lobby.Name,
	})
	if err != nil {
		testLogger.Error("failed to create role", zap.Error(err))
		return err
	}

	_, err = q.CreateDuty(ctx, entities.CreateDutyParams{
		ID:          "jim-the-user",
		RoleID:      jimLobbyRole.ID,
		Instruction: "You are the user.",
		Model:       "gpt-5",
		StreamMode:  false,
	})
	if err != nil {
		testLogger.Error("failed to create duty", zap.Error(err))
		return err
	}

	return nil
}
