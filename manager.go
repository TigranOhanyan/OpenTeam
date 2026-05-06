package openteam

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/openteam/entities"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	_ "turso.tech/database/tursogo"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS // is this necessary?

func init() {
	goose.SetBaseFS(embedMigrations)
}

// TeamDbFactory provides an S3-backed implementation of the checkpoint manager.
type TeamDbFactory struct {
	TempFolder string
}

// NewTeamDbFactory creates a new S3-backed checkpoint manager.
// It ensures the local temporary directory exists before returning.
func NewTeamDbFactory(
	tempFolder string,
	logger *zap.Logger,
) (m TeamDbFactory, err error) {
	if tempFolder == "" {
		logger.Error("temp folder cannot be empty")
		err = fmt.Errorf("temp folder cannot be empty")
		return
	}

	if err = os.MkdirAll(tempFolder, 0755); err != nil {
		logger.Error("failed to create local storage directory", zap.String("path", tempFolder), zap.Error(err))
		err = fmt.Errorf("failed to create local storage directory '%s': %w", tempFolder, err)
		return
	}

	m = TeamDbFactory{
		TempFolder: tempFolder,
	}

	return
}

type TeamDb struct {
	DB           *sql.DB
	Queries      *entities.Queries
	FileFullPath string
	isClosed     bool
}

func (m *TeamDbFactory) NewTeamDb(
	ctx context.Context,
	fileName string,
	logger *zap.Logger,
) (teamDb *TeamDb, err error) {
	path := filepath.Join(m.TempFolder, fileName)

	logger = logger.With(zap.String("path", path))

	err = cleanup(path, logger)
	if err != nil {
		logger.Error("failed to cleanup", zap.Error(err))
		return
	}

	teamDb, err = openAndMigrate(path, logger)
	if err != nil {
		logger.Error("failed to open and migrate database", zap.Error(err))
		return
	}

	return teamDb, nil
}

func (m *TeamDbFactory) ReconstituteFromIO(
	ctx context.Context,
	reader io.Reader,
	targetFileName string,
	logger *zap.Logger,
) (teamDb *TeamDb, err error) {

	path := filepath.Join(m.TempFolder, targetFileName)

	logger = logger.With(zap.String("path", path))

	err = cleanup(path, logger)
	if err != nil {
		logger.Error("failed to cleanup", zap.Error(err))
		return
	}

	logger.Info("writing database to file...")
	file, err := os.Create(path)
	if err != nil {
		logger.Error("failed to create file", zap.Error(err))
		return
	}

	defer func() {
		err := file.Close()
		if err != nil {
			logger.Error("failed to close file", zap.Error(err))
		}
	}()

	_, err = io.Copy(file, reader)
	if err != nil {
		logger.Error("failed to write to file", zap.Error(err))
		return
	}

	teamDb, err = openAndMigrate(path, logger)
	if err != nil {
		logger.Error("failed to open and migrate database", zap.Error(err))
		return
	}

	return
}

func (m *TeamDbFactory) ReconstituteFromFile(
	ctx context.Context,
	sourceFileName string,
	targetFileName string,
	logger *zap.Logger,
) (teamDb *TeamDb, err error) {

	sourceFilePath := filepath.Join(m.TempFolder, sourceFileName)
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		logger.Error("failed to open source file", zap.Error(err))
		return
	}
	defer sourceFile.Close()
	teamDb, err = m.ReconstituteFromIO(ctx, sourceFile, targetFileName, logger)
	return
}

func openAndMigrate(path string, logger *zap.Logger) (teamdb *TeamDb, err error) {
	logger.Info("opening and migrating sqlite db...")

	db, err := sql.Open("turso", path)
	if err != nil {
		logger.Error("failed opening connection to sqlite", zap.Error(err))
		return nil, err
	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		logger.Error("failed to set goose dialect", zap.Error(err))
		db.Close()
		return nil, err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		logger.Error("failed running migrations", zap.Error(err))
		db.Close()
		return nil, err
	}

	teamdb = &TeamDb{
		DB:           db,
		Queries:      entities.New(db),
		FileFullPath: path,
	}

	return
}

func cleanup(path string, logger *zap.Logger) (err error) {
	logger.Info("checking if db already exists...")
	if _, err = os.Stat(path); err == nil {
		logger.Info("db already exists, deleting...")
		if err = os.Remove(path); err != nil {
			logger.Error("failed to delete existing db", zap.Error(err))
			return
		}

		logger.Info("db deleted")
		return
	}

	logger.Info("db does not exist, skipping cleanup")
	err = nil

	return
}

func (m *TeamDb) Flush() error {
	// Force a WAL checkpoint to ensure all data is written to the main .db file
	// so that it can be safely synced to S3.
	_, err := m.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	return err
}

func (m *TeamDb) Close() error {
	if m.isClosed {
		return nil
	}
	m.isClosed = true
	if err := m.Flush(); err != nil {
		// We log but don't return here so we still attempt to close the DB
		fmt.Printf("failed to checkpoint WAL: %v\n", err)
	}
	return m.DB.Close()
}
