// Package repository provides data access layer for biometric data persistence.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/MAMUER/Project/internal/domain"
	"github.com/google/uuid"
)

type BiometricRepository interface {
	Save(ctx context.Context, data *domain.BiometricData) error
	GetByUser(ctx context.Context, userID string, limit int) ([]*domain.BiometricData, error)
	GetLatest(ctx context.Context, userID, metricType string) (*domain.BiometricData, error)
}

type biometricRepository struct {
	db *sql.DB
}

func NewBiometricRepository(db *sql.DB) BiometricRepository {
	return &biometricRepository{db: db}
}

func (r *biometricRepository) Save(ctx context.Context, data *domain.BiometricData) error {
	id := uuid.New().String()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, data.UserID, data.MetricType, data.Value, data.Timestamp, data.DeviceType)
	return err
}

func (r *biometricRepository) GetByUser(ctx context.Context, userID string, limit int) ([]*domain.BiometricData, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, metric_type, value, timestamp, device_type
		FROM biometric_data
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []*domain.BiometricData
	for rows.Next() {
		var data domain.BiometricData
		var timestamp time.Time
		if err := rows.Scan(&data.ID, &data.UserID, &data.MetricType, &data.Value, &timestamp, &data.DeviceType); err != nil {
			return nil, err
		}
		data.Timestamp = timestamp
		results = append(results, &data)
	}
	return results, rows.Err()
}

func (r *biometricRepository) GetLatest(ctx context.Context, userID, metricType string) (*domain.BiometricData, error) {
	var data domain.BiometricData
	var timestamp time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, metric_type, value, timestamp, device_type
		FROM biometric_data
		WHERE user_id = $1 AND metric_type = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`, userID, metricType).Scan(&data.ID, &data.UserID, &data.MetricType, &data.Value, &timestamp, &data.DeviceType)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	data.Timestamp = timestamp
	return &data, nil
}
