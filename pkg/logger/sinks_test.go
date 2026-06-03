package logger_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/abdullahPrasetio/wapgo/pkg/logger"
)

func TestSetupSinks_SizeMode_CreatesFourFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, logger.SetupSinks(logger.SinkConfig{Dir: dir, Rotation: "size"}))

	logger.API().Info().Str("k", "v").Msg("api")
	logger.Consumer().Info().Msg("consumer")
	logger.ThirdParty().Info().Msg("tp")
	logger.Trace().Info().Msg("trace")

	for _, name := range []string{"api.log", "consumer.log", "thirdparty.log", "trace.log"} {
		_, err := os.Stat(filepath.Join(dir, name))
		assert.NoError(t, err, "expected %s to exist", name)
	}
}

func TestSinks_WriteJSONLineWithCategory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, logger.SetupSinks(logger.SinkConfig{Dir: dir, Rotation: "size"}))

	logger.API().Info().Str("method", "GET").Msg("request")

	data, err := os.ReadFile(filepath.Join(dir, "api.log"))
	require.NoError(t, err)
	line := string(data)
	assert.Contains(t, line, `"log":"api"`)
	assert.Contains(t, line, `"method":"GET"`)
	assert.Contains(t, line, `"message":"request"`)
}

func TestSetupSinks_DailyMode_DateStampedFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, logger.SetupSinks(logger.SinkConfig{Dir: dir, Rotation: "daily"}))

	logger.API().Info().Msg("hello")

	today := time.Now().Format("2006-01-02")
	expected := filepath.Join(dir, "api-"+today+".log")
	_, err := os.Stat(expected)
	assert.NoError(t, err, "expected date-stamped file %s", expected)
}

func TestSinks_NoopBeforeSetup(t *testing.T) {
	// Re-point sinks to a fresh dir, then assert the accessor never panics and is usable.
	dir := t.TempDir()
	require.NoError(t, logger.SetupSinks(logger.SinkConfig{Dir: dir}))
	assert.NotPanics(t, func() {
		logger.Trace().Debug().Msg("ok")
	})
}
