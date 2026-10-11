package middleware

import (
	"testing"

	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// newObserverCore возвращает in-memory ядро логов. Записанные логи сбрасываются
// после завершения теста, поэтому helper не возвращает сам ObservedLogs.
func newObserverCore(t *testing.T, level zapcore.Level) zapcore.Core {
	t.Helper()
	core, recorded := observer.New(level)
	t.Cleanup(func() { recorded.TakeAll() })
	return core
}
