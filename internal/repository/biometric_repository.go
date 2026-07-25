// Package repository provides data access layer for biometric data persistence.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/MAMUER/project/internal/domain"
)

// BiometricRepository defines persistence operations for biometric data.
type BiometricRepository interface {
	Save(ctx context.Context, data *domain.BiometricData) error
	GetByUser(ctx context.Context, userID string, limit int) ([]*domain.BiometricData, error)
	GetLatest(ctx context.Context, userID, metricType string) (*domain.BiometricData, error)
}

type biometricRepository struct {
	db *sql.DB
}

// NewBiometricRepository creates a new BiometricRepository.
func NewBiometricRepository(db *sql.DB) BiometricRepository {
	return &biometricRepository{db: db}
}

// Save persists biometric data. If data.ID is empty, a new UUID is generated.
func (r *biometricRepository) Save(ctx context.Context, data *domain.BiometricData) error {
	if data == nil {
		return errors.New("biometric data is nil")
	}
	if data.UserID == "" {
		return errors.New("user_id is required")
	}
	if data.MetricType == "" {
		return errors.New("metric_type is required")
	}

	if data.ID == "" {
		data.ID = uuid.New().String()
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, data.ID, data.UserID, data.MetricType, data.Value, data.Timestamp, data.DeviceType, data.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert biometric data: %w", err)
	}
	return nil
}

// GetByUser returns biometric records for a user ordered by timestamp descending.
func (r *biometricRepository) GetByUser(ctx context.Context, userID string, limit int) ([]*domain.BiometricData, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if limit < 0 {
		return nil, errors.New("limit must be non-negative")
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
		FROM biometric_data
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("query biometric data by user: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var results []*domain.BiometricData
	for rows.Next() {
		var data domain.BiometricData
		var timestamp, createdAt time.Time
		if scanErr := rows.Scan(&data.ID, &data.UserID, &data.MetricType, &data.Value, &timestamp, &data.DeviceType, &createdAt); scanErr != nil {
			return nil, fmt.Errorf("scan biometric data row: %w", scanErr)
		}
		data.Timestamp = timestamp
		data.CreatedAt = createdAt
		results = append(results, &data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate biometric rows: %w", err)
	}
	return results, nil
}

// GetLatest returns the most recent biometric record for a user and metric type.
// Returns sql.ErrNoRows if no matching record exists.
func (r *biometricRepository) GetLatest(ctx context.Context, userID, metricType string) (*domain.BiometricData, error) {
	if userID == "" {
		return nil, errors.New("user_id is required")
	}
	if metricType == "" {
		return nil, errors.New("metric_type is required")
	}

	var data domain.BiometricData
	var timestamp, createdAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, metric_type, value, timestamp, device_type, created_at
		FROM biometric_data
		WHERE user_id = $1 AND metric_type = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`, userID, metricType).Scan(&data.ID, &data.UserID, &data.MetricType, &data.Value, &timestamp, &data.DeviceType, &createdAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("query latest biometric: %w", err)
	}
	data.Timestamp = timestamp
	data.CreatedAt = createdAt
	return &data, nil
}
