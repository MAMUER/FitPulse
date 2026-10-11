package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newObserverCore возвращает in-memory ядро логов уровня Info. Записанные логи
// сбрасываются после завершения теста, поэтому helper не возвращает ObservedLogs.
func newObserverCore(t *testing.T) zapcore.Core {
	t.Helper()
	core, recorded := observer.New(zap.InfoLevel)
	t.Cleanup(func() { recorded.TakeAll() })
	return core
}
