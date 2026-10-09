// Package pgx provides PostgreSQL repository implementations using pgx.
package pgx

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/repository/shared"
)

// BiometricRepositoryPGX implements biometric operations using pgxpool.Pool.
type BiometricRepositoryPGX struct {
	db DB
}

func NewBiometricRepositoryPGX(db DB) *BiometricRepositoryPGX {
	return &BiometricRepositoryPGX{db: db}
}

func (r *BiometricRepositoryPGX) Create(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {

	err := r.db.QueryRow(ctx, shared.BiometricInsertCreateQuery,
		record.ID, record.UserID, record.MetricType, record.Value, record.Timestamp, record.DeviceType, record.Source,
	).Scan(&record.ID)

	if err != nil {

		return nil, apperrors.Internal("failed to insert biometric record", err)

	}

	return record, nil

}

func (r *BiometricRepositoryPGX) BatchCreate(ctx context.Context, records []*entity.BiometricRecord) (int, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, apperrors.Internal("failed to begin transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := shared.BiometricBatchCreateQuery

	inserted := 0
	for _, rec := range records {
		if ctx.Err() != nil {
			return 0, apperrors.Internal("request canceled", ctx.Err())
		}

		result, err := tx.Exec(ctx, query,
			rec.ID, rec.UserID, rec.MetricType, rec.Value, rec.Timestamp, rec.DeviceType, rec.Source,
		)
		if err != nil {
			return 0, apperrors.Internal("failed to insert biometric record", err)
		}
		if result.RowsAffected() > 0 {
			inserted++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, apperrors.Internal("failed to commit transaction", err)
	}
	return inserted, nil
}

func (r *BiometricRepositoryPGX) GetByUserID(ctx context.Context, userID, metricType string, limit, offset int) ([]*entity.BiometricRecord, error) {

	query, args := shared.BuildBiometricQuery(userID, metricType, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)

	if err != nil {

		return nil, apperrors.Internal("failed to get biometric records", err)

	}

	defer rows.Close()

	return shared.ScanBiometricRows(rows)

}

func (r *BiometricRepositoryPGX) GetLatest(ctx context.Context, userID, metricType string) (*entity.BiometricRecord, error) {

	record := &entity.BiometricRecord{}

	err := r.db.QueryRow(ctx, shared.BiometricGetLatestQuery, userID, metricType).Scan(

		&record.ID, &record.UserID, &record.MetricType, &record.Value,

		&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {

			return nil, apperrors.NotFound("no records found")

		}

		return nil, apperrors.Internal("failed to get latest biometric record", err)

	}

	return record, nil

}

func (r *BiometricRepositoryPGX) Update(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {

	err := r.db.QueryRow(ctx, shared.BiometricUpdateQuery,

		record.Value, record.Timestamp, record.DeviceType, record.ID,
	).Scan(

		&record.ID, &record.UserID, &record.MetricType, &record.Value,

		&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {

			return nil, apperrors.NotFound("record not found")

		}

		return nil, apperrors.Internal("failed to update biometric record", err)

	}

	return record, nil

}

func (r *BiometricRepositoryPGX) Delete(ctx context.Context, id string) error {

	_, err := r.db.Exec(ctx, shared.BiometricDeleteQuery, id)

	if err != nil {

		return apperrors.Internal("failed to delete biometric record", err)

	}

	return nil

}

func (r *BiometricRepositoryPGX) GetByID(ctx context.Context, id string) (*entity.BiometricRecord, error) {
	query := `
		SELECT id, user_id, metric_type, value, timestamp, device_type, source, created_at
		FROM biometric_data WHERE id = $1
	`
	record := &entity.BiometricRecord{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&record.ID, &record.UserID, &record.MetricType, &record.Value,
		&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("record not found")
		}
		return nil, apperrors.Internal("failed to get biometric record", err)
	}
	return record, nil
}

func (r *BiometricRepositoryPGX) CountByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM biometric_data WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, apperrors.Internal("failed to count biometric records", err)
	}
	return count, nil
}

func (r *BiometricRepositoryPGX) DeleteByUser(ctx context.Context, userID string) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM biometric_data WHERE user_id = $1`, userID)
	if err != nil {
		return 0, apperrors.Internal("failed to delete biometric records", err)
	}
	return result.RowsAffected(), nil
}

func (r *BiometricRepositoryPGX) GetMetricsSummary(ctx context.Context, userID string) (map[string]interface{}, error) {
	query := `
		SELECT metric_type, COUNT(*) as count, MIN(timestamp) as first_seen, MAX(timestamp) as last_seen
		FROM biometric_data
		WHERE user_id = $1
		GROUP BY metric_type
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to get metrics summary", err)
	}
	defer rows.Close()

	summary := make(map[string]interface{})
	for rows.Next() {
		var metricType string
		var count int
		var firstSeen, lastSeen time.Time
		if err := rows.Scan(&metricType, &count, &firstSeen, &lastSeen); err != nil {
			return nil, apperrors.Internal("failed to scan metrics summary", err)
		}
		summary[metricType] = map[string]interface{}{
			"count":      count,
			"first_seen": firstSeen,
			"last_seen":  lastSeen,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate metrics summary", err)
	}
	return summary, nil
}
