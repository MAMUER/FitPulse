package main

import (
	"database/sql"
	"encoding/json"
	"sync"

	// Registers the "pgx" database/sql driver (sql.Open("pgx", ...)).
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresDeltaStore хранит персональные дельты Δ_u в PostgreSQL.
// Без него дельты живут только в RAM и теряются при рестарте пода — то есть
// персональное дообучение «на конкретном пользователе» не сохраняется.
type PostgresDeltaStore struct {
	db   *sql.DB
	mu   sync.Mutex
	size int
}

func NewPostgresDeltaStore(db *sql.DB) *PostgresDeltaStore {
	return &PostgresDeltaStore{db: db}
}

// GetDelta возвращает сохранённую дельту пользователя.
func (s *PostgresDeltaStore) GetDelta(userID string) ([]float64, bool) {
	if s == nil || s.db == nil || userID == "" {
		return nil, false
	}

	var payload []byte
	err := s.db.QueryRow(
		`SELECT delta FROM classifier_deltas WHERE user_id = $1`, userID,
	).Scan(&payload)
	if err != nil {
		return nil, false
	}

	var delta []float64
	if err := json.Unmarshal(payload, &delta); err != nil {
		return nil, false
	}
	s.mu.Lock()
	s.size = len(delta)
	s.mu.Unlock()
	return delta, true
}

// SetDelta сохраняет дельту пользователя (upsert).
func (s *PostgresDeltaStore) SetDelta(userID string, delta []float64) {
	if s == nil || s.db == nil || userID == "" || len(delta) == 0 {
		return
	}

	payload, err := json.Marshal(delta)
	if err != nil {
		return
	}

	s.mu.Lock()
	s.size = len(delta)
	s.mu.Unlock()

	_, _ = s.db.Exec(`
		INSERT INTO classifier_deltas (user_id, delta, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id)
		DO UPDATE SET delta = EXCLUDED.delta, updated_at = NOW()
	`, userID, payload)
}

// ResetDelta удаляет дельту пользователя.
func (s *PostgresDeltaStore) ResetDelta(userID string) {
	if s == nil || s.db == nil || userID == "" {
		return
	}
	_, _ = s.db.Exec(`DELETE FROM classifier_deltas WHERE user_id = $1`, userID)
}

// DeltaSize возвращает последний известный размер дельты.
func (s *PostgresDeltaStore) DeltaSize() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size
}

// GetAllDeltas возвращает все дельты — используется /classify/delta-info.
func (s *PostgresDeltaStore) GetAllDeltas() map[string][]float64 {
	out := make(map[string][]float64)
	if s == nil || s.db == nil {
		return out
	}

	rows, err := s.db.Query(`SELECT user_id, delta FROM classifier_deltas`)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var userID string
		var payload []byte
		if err := rows.Scan(&userID, &payload); err != nil {
			continue
		}
		var delta []float64
		if err := json.Unmarshal(payload, &delta); err != nil {
			continue
		}
		out[userID] = delta
	}
	// rows.Err() обязателен: без него обрыв чтения (например, разрыв
	// соединения) выглядит как успешный пустой результат.
	if err := rows.Err(); err != nil {
		return out
	}
	return out
}
