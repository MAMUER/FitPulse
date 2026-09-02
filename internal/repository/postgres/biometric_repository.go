package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type BiometricRepository struct {
	db *sql.DB
}

func NewBiometricRepository(db *sql.DB) port.BiometricRepository {
	return &BiometricRepository{db: db}
}

func (r *BiometricRepository) Create(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {
	query := `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, metric_type, timestamp, source) DO UPDATE SET id = biometric_data.id
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		record.ID, record.UserID, record.MetricType, record.Value, record.Timestamp, record.DeviceType, record.Source,
	).Scan(&record.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to insert biometric record", err)
	}
	return record, nil
}

func (r *BiometricRepository) BatchCreate(ctx context.Context, records []*entity.BiometricRecord) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, apperrors.Internal("failed to begin transaction", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id, metric_type, timestamp, source) DO NOTHING
	`

	inserted := 0
	for _, rec := range records {
		if ctx.Err() != nil {
			return 0, apperrors.Internal("request canceled", ctx.Err())
		}

		result, err := tx.ExecContext(ctx, query,
			rec.ID, rec.UserID, rec.MetricType, rec.Value, rec.Timestamp, rec.DeviceType, rec.Source,
		)
		if err != nil {
			return 0, apperrors.Internal("failed to insert biometric record", err)
		}
		if n, _ := result.RowsAffected(); n > 0 {
			inserted++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, apperrors.Internal("failed to commit transaction", err)
	}
	return inserted, nil
}

func (r *BiometricRepository) GetByUserID(ctx context.Context, userID, metricType string, limit, offset int) ([]*entity.BiometricRecord, error) {
	query := `
		SELECT id, user_id, metric_type, value, timestamp, device_type, source, created_at
		FROM biometric_data WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if metricType != "" {
		argCount++
		query += fmt.Sprintf(" AND metric_type = $%d", argCount)
		args = append(args, metricType)
	}

	argCount++
	query += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT $%d", argCount)
	args = append(args, limit)

	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal("failed to get biometric records", err)
	}
	defer rows.Close()

	var records []*entity.BiometricRecord
	for rows.Next() {
		record := &entity.BiometricRecord{}
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.MetricType, &record.Value,
			&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan biometric record", err)
		}
		records = append(records, record)
	}
	return records, nil
}

func (r *BiometricRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM biometric_data WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Internal("failed to delete biometric record", err)
	}
	return nil
}
