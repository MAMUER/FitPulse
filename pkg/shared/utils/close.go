// Package utils provides shared utility functions.
package utils

import (
	"database/sql"
	"fmt"
	"io"
)

// CloseIgnore закрывает ресурс, игнорируя ошибку (для defer в тестах)
func CloseIgnore(c io.Closer) {
	if c != nil {
		_ = c.Close()
	}
}

// CloseDB безопасно закрывает соединение с БД
func CloseDB(db *sql.DB, log func(string, ...interface{})) {
	if db == nil {
		return
	}
	if err := db.Close(); err != nil {
		if log != nil {
			log("Failed to close database: %v", err)
		}
	}
}

// CloseRows безопасно закрывает rows
func CloseRows(rows *sql.Rows, log func(string, ...interface{})) {
	if rows == nil {
		return
	}
	if err := rows.Close(); err != nil {
		if log != nil {
			log("Failed to close rows: %v", err)
		}
	}
}

// CloseSafe универсальная функция для безопасного закрытия любого io.Closer
func CloseSafe(c io.Closer, log func(string, ...interface{})) {
	if c == nil {
		return
	}
	if err := c.Close(); err != nil {
		if log != nil {
			log("Failed to close resource: %v", err)
		}
	}
}

// CloseWithError закрывает ресурс и возвращает ошибку для обработки
func CloseWithError(c io.Closer) error {
	if c == nil {
		return nil
	}
	err := c.Close()
	if err == nil {
		return nil
	}
	return fmt.Errorf("close resource: %w", err)
}

// MultiCloser позволяет закрыть несколько ресурсов последовательно
type MultiCloser []io.Closer

// Close закрывает все ресурсы в порядке добавления
func (mc MultiCloser) Close() error {
	var errs []error
	for _, c := range mc {
		if c != nil {
			if err := c.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("multi-close errors: %v", errs)
	}
	return nil
}

// Add добавляет новый ресурс в список для закрытия
func (mc *MultiCloser) Add(c io.Closer) {
	if c != nil {
		*mc = append(*mc, c)
	}
}
