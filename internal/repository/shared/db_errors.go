package shared

import (
	"database/sql"
	"errors"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/logger"
)

var dbLog = logger.New("repository.postgres")

// CloseRows закрывает результат запроса, логируя ошибку закрытия.
// Ошибка Close после полного чтения набора строк недействительна для вызывающего
// кода, поэтому её нельзя вернуть, но и терять молчать нельзя.
func CloseRows(rows *sql.Rows) {
	if closeErr := rows.Close(); closeErr != nil {
		dbLog.Warn("failed to close result set", zap.Error(closeErr))
	}
}

// RollbackTx откатывает транзакцию и логирует ошибку.
// sql.ErrTxDone означает, что транзакция уже зафиксирована или откачена,
// и не является ошибкой.
func RollbackTx(tx *sql.Tx) {
	if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
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
