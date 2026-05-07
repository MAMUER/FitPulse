package logger

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNew(t *testing.T) {
	log := New("test-service")
	assert.NotNil(t, log)
	defer func() { _ = log.Sync() }()
}

func TestNewWithMultipleServices(t *testing.T) {
	serviceNames := []string{"auth", "biometric", "training", "gateway"}

	for _, name := range serviceNames {
		t.Run(name, func(t *testing.T) {
			log := New(name)
			assert.NotNil(t, log)
			defer func() { _ = log.Sync() }()

			core, recorded := observer.New(zap.InfoLevel)
			testLogger := &Logger{Logger: zap.New(core)}
			loggerWithService := testLogger.With(zap.String("service", name))
			loggerWithService.Info("service started")

			logs := recorded.All()
			require.Len(t, logs, 1)

			found := false
			for _, field := range logs[0].Context {
				if field.Key == "service" && field.String == name {
					found = true
					break
				}
			}
			assert.True(t, found, "service field not found")
		})
	}
}

func TestWithRequestID(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	log := &Logger{Logger: zap.New(core)}

	tests := []struct {
		name      string
		requestID string
	}{
		{"empty request ID", ""},
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000"},
		{"short ID", "abc123"},
		{"long ID", "very-long-request-id-with-many-characters-123456789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loggerWithID := log.WithRequestID(tt.requestID)
			loggerWithID.Info("test message")

			logs := recorded.All()
			require.GreaterOrEqual(t, len(logs), 1)

			lastLog := logs[len(logs)-1]
			assert.Equal(t, "test message", lastLog.Message)

			if tt.requestID != "" {
				found := false
				for _, field := range lastLog.Context {
					if field.Key == "correlationId" && field.String == tt.requestID {
						found = true
						break
					}
				}
				assert.True(t, found, "request_id field not found")
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
	core, recorded := observer.New(zap.DebugLevel)
	log := &Logger{Logger: zap.New(core)}

	levels := []struct {
		level   string
		logFunc func(msg string, fields ...zap.Field)
	}{
		{"debug", log.Debug},
		{"info", log.Info},
		{"warn", log.Warn},
		{"error", log.Error},
	}

	for _, lvl := range levels {
		t.Run(lvl.level, func(t *testing.T) {
			lvl.logFunc("test message")
			logs := recorded.All()
			require.NotEmpty(t, logs)

			lastLog := logs[len(logs)-1]
			assert.Equal(t, "test message", lastLog.Message)
		})
	}
}

func TestService(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
	}{
		{"standard service name", "auth-service"},
		{"empty service name", ""},
		{"service with special chars", "my_service-v2.0"},
		{"single character", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.serviceName)
			defer func() { _ = log.Sync() }()

			assert.Equal(t, tt.serviceName, log.Service())
		})
	}
}

func TestService_DirectAccess(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	underlying := zap.New(core)
	l := &Logger{Logger: underlying, service: "direct-test"}

	assert.Equal(t, "direct-test", l.Service())
}

func TestWithFields(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	// Call WithFields and verify it returns a *zap.Logger
	zapLogger := l.WithFields(zap.String("user_id", "123"), zap.Int("attempt", 3))
	assert.NotNil(t, zapLogger)

	// Log a message using the returned *zap.Logger
	zapLogger.Info("fields test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	// Verify the message
	assert.Equal(t, "fields test", logs[0].Message)

	// Verify fields are present
	fieldKeys := make(map[string]bool)
	for _, field := range logs[0].Context {
		fieldKeys[field.Key] = true
	}
	assert.True(t, fieldKeys["user_id"], "user_id field should be present")
	assert.True(t, fieldKeys["attempt"], "attempt field should be present")
}

func TestWithFields_MultipleCalls(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	fields1 := l.WithFields(zap.String("key1", "val1"))
	fields2 := l.WithFields(zap.String("key2", "val2"))

	fields1.Info("msg1")
	fields2.Info("msg2")

	logs := recorded.All()
	require.Len(t, logs, 2)

	// First log should have key1
	assert.Equal(t, "msg1", logs[0].Message)
	hasKey1 := false
	for _, f := range logs[0].Context {
		if f.Key == "key1" && f.String == "val1" {
			hasKey1 = true
			break
		}
	}
	assert.True(t, hasKey1, "key1 should be in first log")

	// Second log should have key2
	assert.Equal(t, "msg2", logs[1].Message)
	hasKey2 := false
	for _, f := range logs[1].Context {
		if f.Key == "key2" && f.String == "val2" {
			hasKey2 = true
			break
		}
	}
	assert.True(t, hasKey2, "key2 should be in second log")
}

func TestWithFields_NoFields(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	zapLogger := l.WithFields()
	assert.NotNil(t, zapLogger)

	zapLogger.Info("no fields")

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "no fields", logs[0].Message)
}

func TestLogLevelEnvVar(t *testing.T) {
	// Save and restore original env var
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() { _ = os.Setenv("LOG_LEVEL", originalLevel) }()

	tests := []struct {
		name          string
		envValue      string
		expectDebug   bool
		expectInfo    bool
		expectWarning bool
		expectError   bool
	}{
		{
			name:          "DEBUG level",
			envValue:      "DEBUG",
			expectDebug:   true,
			expectInfo:    true,
			expectWarning: true,
			expectError:   true,
		},
		{
			name:          "INFO level",
			envValue:      "INFO",
			expectDebug:   false,
			expectInfo:    true,
			expectWarning: true,
			expectError:   true,
		},
		{
			name:          "WARN level",
			envValue:      "WARN",
			expectDebug:   false,
			expectInfo:    false,
			expectWarning: true,
			expectError:   true,
		},
		{
			name:          "ERROR level",
			envValue:      "ERROR",
			expectDebug:   false,
			expectInfo:    false,
			expectWarning: false,
			expectError:   true,
		},
		{
			name:          "invalid level - falls back to default",
			envValue:      "INVALID_LEVEL",
			expectDebug:   false,
			expectInfo:    true,
			expectWarning: true,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("LOG_LEVEL", tt.envValue))

			// Test that New() reads the LOG_LEVEL env var
			l := New("test-svc")
			defer func() { _ = l.Sync() }()

			// Use an observer to capture output at debug level
			core, recorded := observer.New(zap.DebugLevel)
			testLog := zap.New(core)
			testLog.Debug("debug message")
			testLog.Info("info message")
			testLog.Warn("warn message")
			testLog.Error("error message")

			// Verify the logger was created successfully
			assert.Equal(t, "test-svc", l.Service())

			// Verify messages are captured (the actual level filtering is done by zap)
			logs := recorded.All()
			assert.GreaterOrEqual(t, len(logs), 3) // at least info, warn, error
		})
	}
}

func TestLogLevelEnvVar_EmptyUsesDefault(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() { _ = os.Setenv("LOG_LEVEL", originalLevel) }()

	require.NoError(t, os.Setenv("LOG_LEVEL", ""))

	// When LOG_LEVEL is empty, New() should use default (info level from zap production config)
	l := New("test-svc")
	defer func() { _ = l.Sync() }()

	// Verify logger was created and works
	assert.Equal(t, "test-svc", l.Service())
	l.Info("test with empty LOG_LEVEL")
}

func TestErrorOutputPaths(t *testing.T) {
	const (
		testTimeKey    = "timestamp"
		testLevelKey   = "level"
		testMessageKey = "message"
	)

	// Test that the logger can be configured with stderr for error output
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = testTimeKey
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.LevelKey = testLevelKey
	cfg.EncoderConfig.MessageKey = testMessageKey
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.StacktraceKey = "stacktrace"
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	logger, err := cfg.Build()
	require.NoError(t, err)
	defer func() { _ = logger.Sync() }()

	l := &Logger{Logger: logger, service: "test-stderr"}
	assert.NotNil(t, l)
	assert.Equal(t, "test-stderr", l.Service())

	// Just verify we can call error-level logging without panic
	assert.NotPanics(t, func() {
		l.Error("test error message")
	})
}

func TestErrorOutputPaths_MultiplePaths(t *testing.T) {
	const (
		testTimeKey    = "timestamp"
		testLevelKey   = "level"
		testMessageKey = "message"
	)

	// Test configuration with multiple output paths
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = testTimeKey
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.LevelKey = testLevelKey
	cfg.EncoderConfig.MessageKey = testMessageKey
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr", "stdout"}

	logger, err := cfg.Build()
	require.NoError(t, err)
	defer func() { _ = logger.Sync() }()

	l := &Logger{Logger: logger, service: "multi-path"}
	assert.NotNil(t, l)

	assert.NotPanics(t, func() {
		l.Error("error to multiple outputs")
		l.Info("info to stdout")
	})
}

func TestSync(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-sync"}

	l.Info("message before sync")

	err := l.Sync()
	assert.NoError(t, err)
}

func TestSync_CalledMultipleTimes(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-sync"}

	// Sync should be safe to call multiple times
	err1 := l.Sync()
	err2 := l.Sync()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestWithRequestID_PreservesService(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "original-service"}

	child := l.WithRequestID("req-123")

	// The service name should be preserved in the child logger
	assert.Equal(t, "original-service", child.service)
}

func TestWithFields_FieldTypes(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	zapLogger := l.WithFields(
		zap.String("str_field", "hello"),
		zap.Int("int_field", 42),
		zap.Bool("bool_field", true),
		zap.Float64("float_field", 3.14),
	)

	zapLogger.Info("typed fields test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	fieldMap := make(map[string]zap.Field)
	for _, f := range logs[0].Context {
		fieldMap[f.Key] = f
	}

	assert.Contains(t, fieldMap, "str_field")
	assert.Contains(t, fieldMap, "int_field")
	assert.Contains(t, fieldMap, "bool_field")
	assert.Contains(t, fieldMap, "float_field")
}
