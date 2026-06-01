package logger_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/abdullahPrasetio/wapgo/pkg/logger"
)

func TestSetup_Development(t *testing.T) {
	// Should not panic
	logger.Setup("development", "debug", "logs/test.log", false, "test-service")
}

func TestSetup_Production(t *testing.T) {
	logger.Setup("production", "info", "", false, "test-service")
}

func TestSetup_InvalidLevel_FallsBackToInfo(t *testing.T) {
	logger.Setup("development", "notALevel", "", false, "svc")
}

func TestWithRequestID_And_FromContext(t *testing.T) {
	ctx := context.Background()
	ctx = logger.WithRequestID(ctx, "req-123")
	log := logger.FromContext(ctx)
	assert.NotNil(t, log)
}

func TestRequestIDFromContext_Present(t *testing.T) {
	ctx := logger.WithRequestID(context.Background(), "abc-456")
	assert.Equal(t, "abc-456", logger.RequestIDFromContext(ctx))
}

func TestRequestIDFromContext_Absent(t *testing.T) {
	assert.Equal(t, "", logger.RequestIDFromContext(context.Background()))
}

func TestFromContext_NoRequestID(t *testing.T) {
	log := logger.FromContext(context.Background())
	assert.NotNil(t, log)
}
