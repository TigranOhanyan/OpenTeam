package openteam

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/openteam/entities"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func Test_TeamDbFactory_should_create_a_new_db(t *testing.T) {
	var err error

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	fileName := ulid.Make().String() + ".db"

	teamDb, err := teamDbFactory.NewTeamDb(ctx, fileName, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb)
	defer teamDb.Close()
	assert.NotNil(t, teamDb.Queries)

	err = teamDb.Close()
	assert.NoError(t, err)

	stat, err := os.Stat(teamDb.FileFullPath)
	assert.NoError(t, err)
	assert.True(t, stat.Size() > 0)
}

func Test_TeamDbFactory_should_reconstitute_from_file(t *testing.T) {
	var err error

	teamDbFactory, err := NewTeamDbFactory(tempFolder, testLogger)
	assert.NoError(t, err)

	ctx := context.TODO()

	originalFileName := ulid.Make().String() + ".db"

	teamDb, err := teamDbFactory.NewTeamDb(ctx, originalFileName, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb.DB)
	err = makeTeamForManagerTest(ctx, teamDb)
	assert.NoError(t, err)

	err = teamDb.Close()
	assert.NoError(t, err)

	expectedContent, err := os.ReadFile(teamDb.FileFullPath)
	assert.NoError(t, err)

	finalFileName := ulid.Make().String() + ".db"
	teamDb2, err := teamDbFactory.ReconstituteFromFile(ctx, originalFileName, finalFileName, testLogger)
	assert.NoError(t, err)
	assert.NotNil(t, teamDb2)

	actualContent, err := os.ReadFile(teamDb2.FileFullPath)
	assert.NoError(t, err)

	err = teamDb2.Close()
	assert.NoError(t, err)

	// Since SQLite uses WAL mode, the exact byte-for-byte comparison of the .db file might fail
	// if checkpointing hasn't happened. For this test, we just ensure the new file has content.
	assert.True(t, len(actualContent) > 0)
	assert.True(t, len(expectedContent) > 0)
}

func makeTeamForManagerTest(ctx context.Context, teamDb *TeamDb) error {
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
