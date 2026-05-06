package testutils

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"go.uber.org/zap"
)

var testDataDir string
var JunkDir string

func init() {
	_, filename, _, _ := runtime.Caller(0)
	helperDir := filepath.Dir(filename)
	testDataDir = filepath.Join(helperDir, "testdata")
	JunkDir = filepath.Join(testDataDir, "junk")

}

func mustReadFromJsonFile(docDir string, logger *zap.Logger) (documentPayload map[string]interface{}) {
	jsonFile, err := os.Open(docDir)
	if err != nil {
		logger.Error("failed to open file", zap.Error(err))
		panic(err)
	}
	defer jsonFile.Close()

	bytes, err := io.ReadAll(jsonFile)
	if err != nil {
		logger.Error("failed to read file", zap.Error(err))
		panic(err)
	}

	err = json.Unmarshal(bytes, &documentPayload)
	if err != nil {
		logger.Error("failed to unmarshal file", zap.Error(err))
		panic(err)
	}

	return
}
