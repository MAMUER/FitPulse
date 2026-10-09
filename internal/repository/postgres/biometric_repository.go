// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
	"github.com/MAMUER/project/internal/repository/shared"
)

type BiometricRepository struct {
	db *sql.DB
}

func NewBiometricRepository(db *sql.DB) port.BiometricRepository {

	return &BiometricRepository{db: db}

}

func (r *BiometricRepository) Create(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {

	err := r.db.QueryRowContext(ctx, shared.BiometricInsertCreateQuery,

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

	query := shared.BiometricBatchCreateQuery

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

	query, args := shared.BuildBiometricQuery(userID, metricType, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...) //nolint:rowserrcheck // ScanBiometricRows checks rows.Err internally

	if err != nil {

		return nil, apperrors.Internal("failed to get biometric records", err)

	}

	defer func() { _ = rows.Close() }()

	return shared.ScanBiometricRows(rows)

}

func (r *BiometricRepository) Delete(ctx context.Context, id string) error {

	_, err := r.db.ExecContext(ctx, shared.BiometricDeleteQuery, id)

	if err != nil {

		return apperrors.Internal("failed to delete biometric record", err)

	}

	return nil

}

func (r *BiometricRepository) GetLatest(ctx context.Context, userID, metricType string) (*entity.BiometricRecord, error) {

	record := &entity.BiometricRecord{}

	err := r.db.QueryRowContext(ctx, shared.BiometricGetLatestQuery, userID, metricType).Scan(

		&record.ID, &record.UserID, &record.MetricType, &record.Value,

		&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {

			return nil, apperrors.NotFound("no records found")

		}

		return nil, apperrors.Internal("failed to get latest biometric record", err)

	}

	return record, nil

}

func (r *BiometricRepository) Update(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {

	err := r.db.QueryRowContext(ctx, shared.BiometricUpdateQuery,

		record.Value, record.Timestamp, record.DeviceType, record.ID,
	).Scan(

		&record.ID, &record.UserID, &record.MetricType, &record.Value,

		&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
	)

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {

			return nil, apperrors.NotFound("record not found")

		}

		return nil, apperrors.Internal("failed to update biometric record", err)

	}

	return record, nil

}
