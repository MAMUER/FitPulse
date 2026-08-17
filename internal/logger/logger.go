// Package logger provides structured logging utilities.
package logger

import (
	"context"
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	correlationIDKey = "correlation_id"
	userIDKey        = "user_id"
)

// Logger структура для обёртки над zap.Logger
type Logger struct {
	*zap.Logger
	service string
}

// New создает новый логгер с именем сервиса
func New(service string) *Logger {
	return newLogger(service, false, nil)
}

// Development создает логгер для локальной разработки с цветным выводом в консоль
func Development(service string) *Logger {
	return newLogger(service, true, nil)
}

var fatal = log.Fatal

func newLogger(service string, development bool, configOverride *zap.Config) *Logger {
	var cfg zap.Config
	if configOverride != nil {
		cfg = *configOverride
	} else if development {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Encoding = "json"
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		cfg.EncoderConfig.LevelKey = "level"
		cfg.EncoderConfig.MessageKey = "message"
		cfg.EncoderConfig.CallerKey = "caller"
		cfg.EncoderConfig.StacktraceKey = "stacktrace"
	}
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(lvl)); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(level)
		}
	}

	logger, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		fatal("failed to initialize logger", zap.Error(err))
		return nil
	}

	return &Logger{
		Logger:  logger.With(zap.String("service", service)),
		service: service,
	}
}

// Service возвращает имя сервиса для логирования
func (l *Logger) Service() string {
	return l.service
}

func (l *Logger) withFields(fields ...zap.Field) *Logger {
	return &Logger{
		Logger:  l.With(fields...),
		service: l.service,
	}
}

// WithRequestID добавляет correlationId к контексту логгера
func (l *Logger) WithRequestID(correlationID string) *Logger {
	return l.withFields(zap.String("correlationId", correlationID))
}

// WithUserID добавляет userId к контексту логгера
func (l *Logger) WithUserID(userID string) *Logger {
	return l.withFields(zap.String("userId", userID))
}

// WithAction добавляет action к контексту логгера
func (l *Logger) WithAction(action string) *Logger {
	return l.withFields(zap.String("action", action))
}

// WithDuration добавляет durationMs к контексту логгера
func (l *Logger) WithDuration(duration time.Duration) *Logger {
	return l.withFields(zap.Int64("durationMs", duration.Milliseconds()))
}

// WithMetadata добавляет метаданные к контексту логгера
func (l *Logger) WithMetadata(metadata map[string]interface{}) *Logger {
	fields := make([]zap.Field, 0, len(metadata))
	for k, v := range metadata {
		switch val := v.(type) {
		case string:
			fields = append(fields, zap.String(k, val))
		case int:
			fields = append(fields, zap.Int(k, val))
		case int64:
			fields = append(fields, zap.Int64(k, val))
		case float64:
			fields = append(fields, zap.Float64(k, val))
		case bool:
			fields = append(fields, zap.Bool(k, val))
		default:
			fields = append(fields, zap.Any(k, val))
		}
	}
	return l.withFields(fields...)
}

// WithFields добавляет произвольные поля к контексту логгера
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return l.withFields(fields...)
}

// WithCallerSkip возвращает логгер с измененным уровнем пропуска caller'ов
func (l *Logger) WithCallerSkip(skip int) *Logger {
	return &Logger{
		Logger:  l.WithOptions(zap.AddCallerSkip(skip)),
		service: l.service,
	}
}

// FromContext создает логгер с полями из контекста запроса
func FromContext(ctx context.Context, base *Logger) *Logger {
	logger := base

	if cid, ok := ctx.Value(correlationIDKey).(string); ok && cid != "" {
		logger = logger.WithRequestID(cid)
	}
	if uid, ok := ctx.Value(userIDKey).(string); ok && uid != "" {
		logger = logger.WithUserID(uid)
	}

	return logger
}

// Errorw логирует ошибку с дополнительными полями
func (l *Logger) Errorw(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	l.Error(msg, fields...)
}

// Sync гарантирует запись всех буферизированных логов
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
