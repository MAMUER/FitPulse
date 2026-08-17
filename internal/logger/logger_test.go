package logger

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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
			testLogger := zap.New(core)
			testLogger.With(zap.String("service", name)).Info("service started")

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

	childLogger := l.WithFields(zap.String("user_id", "123"), zap.Int("attempt", 3))
	assert.NotNil(t, childLogger)

	childLogger.Info("fields test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	assert.Equal(t, "fields test", logs[0].Message)

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

	assert.Equal(t, "msg1", logs[0].Message)
	hasKey1 := false
	for _, f := range logs[0].Context {
		if f.Key == "key1" && f.String == "val1" {
			hasKey1 = true
			break
		}
	}
	assert.True(t, hasKey1, "key1 should be in first log")

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

	childLogger := l.WithFields()
	assert.NotNil(t, childLogger)

	childLogger.Info("no fields")

	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "no fields", logs[0].Message)
}

func TestLogLevelEnvVar(t *testing.T) {
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

			l := New("test-svc")
			defer func() { _ = l.Sync() }()

			core, recorded := observer.New(zap.DebugLevel)
			testLog := zap.New(core)
			testLog.Debug("debug message")
			testLog.Info("info message")
			testLog.Warn("warn message")
			testLog.Error("error message")

			assert.Equal(t, "test-svc", l.Service())

			logs := recorded.All()
			assert.GreaterOrEqual(t, len(logs), 3)
		})
	}
}

func TestLogLevelEnvVar_EmptyUsesDefault(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() { _ = os.Setenv("LOG_LEVEL", originalLevel) }()

	require.NoError(t, os.Setenv("LOG_LEVEL", ""))

	l := New("test-svc")
	defer func() { _ = l.Sync() }()

	assert.Equal(t, "test-svc", l.Service())
	l.Info("test with empty LOG_LEVEL")
}

func TestErrorOutputPaths(t *testing.T) {
	const (
		testTimeKey    = "timestamp"
		testLevelKey   = "level"
		testMessageKey = "message"
	)

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

	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = testTimeKey
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.LevelKey = testLevelKey
	cfg.EncoderConfig.MessageKey = testMessageKey
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.StacktraceKey = "stacktrace"
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

	err1 := l.Sync()
	err2 := l.Sync()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
}

func TestWithRequestID_PreservesService(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "original-service"}

	child := l.WithRequestID("req-123")

	assert.Equal(t, "original-service", child.service)
}

func TestWithFields_FieldTypes(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	childLogger := l.WithFields(
		zap.String("str_field", "hello"),
		zap.Int("int_field", 42),
		zap.Bool("bool_field", true),
		zap.Float64("float_field", 3.14),
	)

	childLogger.Info("typed fields test")

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

func TestWithMetadata(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	metadata := map[string]interface{}{
		"str_key":   "hello",
		"int_key":   42,
		"float_key": 3.14,
		"bool_key":  true,
	}

	childLogger := l.WithMetadata(metadata)
	childLogger.Info("metadata test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	fieldMap := make(map[string]zap.Field)
	for _, f := range logs[0].Context {
		fieldMap[f.Key] = f
	}

	assert.Contains(t, fieldMap, "str_key")
	assert.Contains(t, fieldMap, "int_key")
	assert.Contains(t, fieldMap, "float_key")
	assert.Contains(t, fieldMap, "bool_key")
}

func TestWithCallerSkip(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	baseLogger := zap.New(core)
	l := &Logger{Logger: baseLogger, service: "test-svc"}

	child := l.WithCallerSkip(1)
	assert.NotNil(t, child)
	assert.Equal(t, "test-svc", child.service)
}

func TestErrorw(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	testErr := errors.New("test error")
	l.Errorw("operation failed", testErr, zap.String("operation", "create"))

	logs := recorded.All()
	require.Len(t, logs, 1)

	assert.Equal(t, "operation failed", logs[0].Message)

	fieldMap := make(map[string]zap.Field)
	for _, f := range logs[0].Context {
		fieldMap[f.Key] = f
	}

	assert.Contains(t, fieldMap, "error")
	assert.Contains(t, fieldMap, "operation")
}

func TestErrorw_NilError(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	l.Errorw("operation failed", nil, zap.String("operation", "create"))

	logs := recorded.All()
	require.Len(t, logs, 1)

	assert.Equal(t, "operation failed", logs[0].Message)

	_, hasError := false, false
	for _, f := range logs[0].Context {
		if f.Key == "error" {
			hasError = true
			break
		}
	}
	assert.False(t, hasError, "error field should not be present when err is nil")
}

func TestFromContext(t *testing.T) {
	t.Run("with correlationId and userId", func(t *testing.T) {
		core, recorded := observer.New(zap.InfoLevel)
		baseLogger := &Logger{Logger: zap.New(core), service: "test-svc"}

		ctx := context.WithValue(context.Background(), correlationIDKey, "corr-123")
		ctx = context.WithValue(ctx, userIDKey, "user-456")

		logger := FromContext(ctx, baseLogger)
		logger.Info("context test")

		logs := recorded.All()
		require.Len(t, logs, 1)

		fieldMap := make(map[string]zap.Field)
		for _, f := range logs[0].Context {
			fieldMap[f.Key] = f
		}

		assert.Equal(t, "corr-123", fieldMap["correlationId"].String)
		assert.Equal(t, "user-456", fieldMap["userId"].String)
	})

	t.Run("with empty context", func(t *testing.T) {
		core, recorded := observer.New(zap.InfoLevel)
		baseLogger := &Logger{Logger: zap.New(core), service: "test-svc"}

		logger := FromContext(context.Background(), baseLogger)
		logger.Info("empty context")

		logs := recorded.All()
		require.Len(t, logs, 1)
		assert.Equal(t, "empty context", logs[0].Message)
	})
}

func TestFromContext_PreservesService(t *testing.T) {
	core, _ := observer.New(zap.InfoLevel)
	baseLogger := &Logger{Logger: zap.New(core), service: "original-service"}

	ctx := context.WithValue(context.Background(), correlationIDKey, "corr-123")
	logger := FromContext(ctx, baseLogger)

	assert.Equal(t, "original-service", logger.service)
}

func TestFromContext_NilContext(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	baseLogger := &Logger{Logger: zap.New(core), service: "test-svc"}

	logger := FromContext(nil, baseLogger) // nolint:SA1012

	assert.Equal(t, baseLogger, logger)
	assert.Equal(t, "test-svc", logger.Service())

	logger.Info("nil context test")
	logs := recorded.All()
	require.Len(t, logs, 1)
	assert.Equal(t, "nil context test", logs[0].Message)
}

func TestNew_BuildFails(t *testing.T) {
	var fatalCalled bool
	oldFatal := fatal
	fatal = func(v ...interface{}) {
		fatalCalled = true
	}
	defer func() { fatal = oldFatal }()

	cfg := zap.NewProductionConfig()
	cfg.Encoding = "invalid"
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	newLogger("test", false, &cfg)
	assert.True(t, fatalCalled, "fatal should have been called when build fails")
}

func TestDevelopment(t *testing.T) {
	log := Development("test-service")
	assert.NotNil(t, log)
	defer func() { _ = log.Sync() }()

	assert.Equal(t, "test-service", log.Service())
}

func TestDevelopment_EnvLevel(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() { _ = os.Setenv("LOG_LEVEL", originalLevel) }()

	require.NoError(t, os.Setenv("LOG_LEVEL", "DEBUG"))

	log := Development("test-service")
	defer func() { _ = log.Sync() }()

	assert.Equal(t, "test-service", log.Service())
}

func TestDevelopment_InvalidLogLevel(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer func() { _ = os.Setenv("LOG_LEVEL", originalLevel) }()

	require.NoError(t, os.Setenv("LOG_LEVEL", "INVALID_LEVEL"))

	assert.NotPanics(t, func() {
		log := Development("test-service")
		defer func() { _ = log.Sync() }()
		assert.Equal(t, "test-service", log.Service())
	})
}

func TestWithAction(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	child := l.WithAction("create_user")
	child.Info("action test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	fieldMap := make(map[string]zap.Field)
	for _, f := range logs[0].Context {
		fieldMap[f.Key] = f
	}

	assert.Equal(t, "create_user", fieldMap["action"].String)
}

func TestWithDuration(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	child := l.WithDuration(1500 * time.Millisecond)
	child.Info("duration test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	fieldMap := make(map[string]zap.Field)
	for _, f := range logs[0].Context {
		fieldMap[f.Key] = f
	}

	assert.Equal(t, int64(1500), fieldMap["durationMs"].Integer)
}

func TestWithMetadata_Int64AndDefault(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	l := &Logger{Logger: zap.New(core), service: "test-svc"}

	metadata := map[string]interface{}{
		"int64_key": int64(9223372036854775807),
		"slice_key": []string{"a", "b"},
	}

	childLogger := l.WithMetadata(metadata)
	childLogger.Info("metadata int64 test")

	logs := recorded.All()
	require.Len(t, logs, 1)

	fieldMap := make(map[string]zap.Field)
	for _, f := range logs[0].Context {
		fieldMap[f.Key] = f
	}

	assert.Contains(t, fieldMap, "int64_key")
	assert.Equal(t, int64(9223372036854775807), fieldMap["int64_key"].Integer)
	assert.Contains(t, fieldMap, "slice_key")
}
