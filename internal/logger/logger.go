// Package logger provides structured logging utilities.
package logger

import (
	"log"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger структура для обёртки над zap.Logger
type Logger struct {
	*zap.Logger
	service string
}

// New создает новый логгер с именем сервиса
func New(service string) *Logger {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "json"
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.MessageKey = "message"
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.EncoderConfig.StacktraceKey = "stacktrace"
	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	// Добавляем уровень логов из переменной окружения
	if lvl := os.Getenv("LOG_LEVEL"); lvl != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(lvl)); err == nil {
			cfg.Level = zap.NewAtomicLevelAt(level)
		}
	}

	logger, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		log.Fatal("failed to initialize logger", zap.Error(err))
	}

	return &Logger{
		Logger:  logger,
		service: service,
	}
}

// Service возвращает имя сервиса для логирования
func (l *Logger) Service() string {
	return l.service
}

// WithRequestID добавляет correlationId к контексту логгера
func (l *Logger) WithRequestID(correlationID string) *Logger {
	return &Logger{
		Logger:  l.With(zap.String("correlationId", correlationID)),
		service: l.service,
	}
}

// WithUserID добавляет userId к контексту логгера
func (l *Logger) WithUserID(userID string) *Logger {
	return &Logger{
		Logger:  l.With(zap.String("userId", userID)),
		service: l.service,
	}
}

// WithAction добавляет action к контексту логгера
func (l *Logger) WithAction(action string) *Logger {
	return &Logger{
		Logger:  l.With(zap.String("action", action)),
		service: l.service,
	}
}

// WithDuration добавляет durationMs к контексту логгера
func (l *Logger) WithDuration(duration time.Duration) *Logger {
	return &Logger{
		Logger:  l.With(zap.Int64("durationMs", duration.Milliseconds())),
		service: l.service,
	}
}

// WithContext добавляет context объект к логгеру
func (l *Logger) WithContext(context map[string]interface{}) *Logger {
	fields := make([]zap.Field, 0, len(context))
	for k, v := range context {
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
	return &Logger{
		Logger:  l.With(fields...),
		service: l.service,
	}
}

// WithFields добавляет произвольные поля к контексту логгера
func (l *Logger) WithFields(fields ...zap.Field) *zap.Logger {
	return l.With(fields...)
}

// Sync гарантирует запись всех буферизированных логов
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
