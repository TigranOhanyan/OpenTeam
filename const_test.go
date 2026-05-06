package openteam

import (
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	testutils "github.com/openteam/testutil"
	"github.com/wiremock/go-wiremock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var wiremockClient = wiremock.NewClient("http://0.0.0.0:18443")
var testLogger = MustCreateZuluTimeLogger()
var tempFolder = testutils.JunkDir

var llmApiKey = "sk-pro-1234567890"

func MustCreateZuluTimeLogger() (logger *zap.Logger) {

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	timeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
	}

	config.EncoderConfig.EncodeTime = timeEncoder

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger
}

var llmOpenAiClient = openai.NewClient(
	option.WithBaseURL("http://localhost:18443/v1"),
	option.WithAPIKey(llmApiKey),
)

var agentProto = Agent{
	LlmClient: &llmOpenAiClient,
}
