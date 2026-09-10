package pgx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

func scanDevices(rows pgx.Rows) ([]*entity.Device, error) {
	return scanSlice(rows, func(d *entity.Device) error {
		return rows.Scan(&d.ID, &d.UserID, &d.DeviceType, &d.DeviceName, &d.IsConnected, &d.LastSync)
	})
}

func scanSlice[T any](rows pgx.Rows, scanFunc func(*T) error) ([]*T, error) {
	var items []*T
	for rows.Next() {
		item := new(T)
		if err := scanFunc(item); err != nil {
			return nil, apperrors.Internal("failed to scan item", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate items", err)
	}
	return items, nil
}

// DeviceRepositoryPGX implements device operations using pgxpool.Pool.
type DeviceRepositoryPGX struct {
	db DB
}

func NewDeviceRepositoryPGX(db DB) *DeviceRepositoryPGX {
	return &DeviceRepositoryPGX{db: db}
}

func (r *DeviceRepositoryPGX) List(ctx context.Context, userID string) ([]*entity.Device, error) {
	query := `
		SELECT id, user_id, device_type, device_name, is_connected, last_sync
		FROM devices WHERE user_id = $1
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to list devices", err)
	}
	defer rows.Close()

	return scanDevices(rows)
}

func (r *DeviceRepositoryPGX) Create(ctx context.Context, device *entity.Device) (*entity.Device, error) {
	query := `
		INSERT INTO devices (id, user_id, device_type, device_name, token, is_connected, last_sync)
		VALUES ($1, $2, $3, $4, $5, true, NOW())
		RETURNING id
	`
	err := r.db.QueryRow(ctx, query,
		device.ID, device.UserID, device.DeviceType, device.DeviceName, device.Token,
	).Scan(&device.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create device", err)
	}
	return device, nil
}

func (r *DeviceRepositoryPGX) Delete(ctx context.Context, userID, deviceID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM devices WHERE user_id = $1 AND id = $2`, userID, deviceID)
	if err != nil {
		return apperrors.Internal("failed to delete device", err)
	}
	return nil
}

func (r *DeviceRepositoryPGX) GetByID(ctx context.Context, userID, deviceID string) (*entity.Device, error) {
	query := `
		SELECT id, user_id, device_type, device_name, is_connected, last_sync
		FROM devices WHERE user_id = $1 AND id = $2
	`
	device := &entity.Device{}
	err := r.db.QueryRow(ctx, query, userID, deviceID).Scan(
		&device.ID, &device.UserID, &device.DeviceType, &device.DeviceName,
		&device.IsConnected, &device.LastSync,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NotFound("device not found")
		}
		return nil, apperrors.Internal("failed to get device", err)
	}
	return device, nil
}

func (r *DeviceRepositoryPGX) UpdateLastSync(ctx context.Context, deviceID string) error {
	_, err := r.db.Exec(ctx, `UPDATE devices SET last_sync = NOW() WHERE id = $1`, deviceID)
	if err != nil {
		return apperrors.Internal("failed to update device last_sync", err)
	}
	return nil
}

func (r *DeviceRepositoryPGX) CountByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM devices WHERE user_id = $1`, userID).Scan(&count)
	if err != nil {
		return 0, apperrors.Internal("failed to count devices", err)
	}
	return count, nil
}

func (r *DeviceRepositoryPGX) ExistsConnected(ctx context.Context, userID, deviceType string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM devices WHERE user_id = $1 AND device_type = $2 AND is_connected = true)`
	err := r.db.QueryRow(ctx, query, userID, deviceType).Scan(&exists)
	if err != nil {
		return false, apperrors.Internal("failed to check device existence", err)
	}
	return exists, nil
}

func (r *DeviceRepositoryPGX) DisconnectAll(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE devices SET is_connected = false WHERE user_id = $1`, userID)
	if err != nil {
		return apperrors.Internal("failed to disconnect devices", err)
	}
	return nil
}

func (r *DeviceRepositoryPGX) ListConnected(ctx context.Context, userID string) ([]*entity.Device, error) {
	query := `
		SELECT id, user_id, device_type, device_name, is_connected, last_sync
		FROM devices WHERE user_id = $1 AND is_connected = true
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to list connected devices", err)
	}
	defer rows.Close()

	return scanDevices(rows)
}

func (r *DeviceRepositoryPGX) BatchCreate(ctx context.Context, devices []*entity.Device) (int, error) {
	if len(devices) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, apperrors.Internal("failed to begin transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query := `
		INSERT INTO devices (id, user_id, device_type, device_name, token, is_connected, last_sync)
		VALUES ($1, $2, $3, $4, $5, true, NOW())
	`

	inserted := 0
	for _, device := range devices {
		if device.ID == "" {
			device.ID = fmt.Sprintf("dev_%d_%d", time.Now().UnixNano(), inserted)
		}
		if device.Token == "" {
			device.Token = fmt.Sprintf("tok_%d_%d", time.Now().UnixNano(), inserted)
		}

		_, err := tx.Exec(ctx, query,
			device.ID, device.UserID, device.DeviceType, device.DeviceName, device.Token,
		)
		if err != nil {
			return 0, apperrors.Internal("failed to create device", err)
		}
		inserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, apperrors.Internal("failed to commit transaction", err)
	}
	return inserted, nil
}

func (r *DeviceRepositoryPGX) DeleteInactive(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `DELETE FROM devices WHERE last_sync < $1 AND is_connected = false`, olderThan)
	if err != nil {
		return 0, apperrors.Internal("failed to delete inactive devices", err)
	}
	return result.RowsAffected(), nil
}
