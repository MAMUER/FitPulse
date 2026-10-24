package shared

import (
	"database/sql"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

var dbLog = logger.New("repository.postgres")

// EmptySurvey returns an empty, incomplete survey result.
// Centralized to avoid duplication across repository implementations.
func EmptySurvey() (map[string]interface{}, bool, *time.Time, error) {
	return map[string]interface{}{}, false, nil, nil
}

// CloseRows закрывает результат запроса, логируя ошибку закрытия.
// Ошибка Close после полного чтения набора строк недействительна для вызывающего
// кода, поэтому её нельзя вернуть, но и терять молчать нельзя.
func CloseRows(rows *sql.Rows) {
	if closeErr := rows.Close(); closeErr != nil {
		dbLog.Warn("failed to close result set", zap.Error(closeErr))
	}
}

// LogTxRollback логирует ошибку отката транзакции.
// Вызывается с результатом tx.Rollback() внутри defer, чтобы сохранить
// диагностику при неявном откате: sql.ErrTxDone означает, что транзакция
// уже зафиксирована или откачена, и не является ошибкой.
func LogTxRollback(rollbackErr error) {
	if !errors.Is(rollbackErr, sql.ErrTxDone) {
		dbLog.Warn("failed to rollback transaction", zap.Error(rollbackErr))
	}
}

// RowsAffected возвращает количество изменённых строк, логируя ошибку.
func RowsAffected(result sql.Result) int64 {
	n, err := result.RowsAffected()
	if err != nil {
		dbLog.Warn("failed to read affected rows count", zap.Error(err))
		return 0
	}
	return n
}
