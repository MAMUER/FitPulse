package main

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MAMUER/project/internal/apperrors"
)

// TestIsNoSuchUser фиксирует контракт, из-за нарушения которого первая
// авторизация через Яндекс падала в 500 "database error": репозиторий отдаёт
// типизированную apperrors.NotFound ("user not found"), а не сырой sql.ErrNoRows.
func TestIsNoSuchUser(t *testing.T) {
	t.Run("raw sql.ErrNoRows", func(t *testing.T) {
		assert.True(t, isNoSuchUser(sql.ErrNoRows))
	})

	t.Run("wrapped sql.ErrNoRows", func(t *testing.T) {
		assert.True(t, isNoSuchUser(errors.Join(errors.New("query failed"), sql.ErrNoRows)))
	})

	t.Run("typed apperrors.NotFound", func(t *testing.T) {
		err := apperrors.NotFound("user not found")
		assert.True(t, isNoSuchUser(err))
	})

	t.Run("other error is not a miss", func(t *testing.T) {
		assert.False(t, isNoSuchUser(errors.New("connection refused")))
	})

	t.Run("nil is not a miss", func(t *testing.T) {
		assert.False(t, isNoSuchUser(nil))
	})
}
