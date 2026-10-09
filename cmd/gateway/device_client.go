package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MAMUER/project/cmd/gateway/ports"
	"github.com/MAMUER/project/internal/apperrors"
)

var ErrDeviceNotFound = apperrors.NotFound("device not found")

type gatewayDeviceClient struct {
	db *sql.DB
}

func newGatewayDeviceClient(db *sql.DB) *gatewayDeviceClient {
	return &gatewayDeviceClient{db: db}
}

func (c *gatewayDeviceClient) CreateOrUpdateDevice(ctx context.Context, userID, deviceType, deviceName, token string) (string, error) {
	id := fmt.Sprintf("dev_%d_%d", time.Now().UnixNano(), len(userID))
	_, err := c.db.ExecContext(ctx,
		`INSERT INTO devices (id, user_id, device_type, device_name, token, is_connected, last_sync)
		 VALUES ($1, $2, $3, $4, $5, true, NOW())`,
		id, userID, deviceType, deviceName, token,
	)
	if err != nil {
		return "", apperrors.Internal("failed to create device", err)
	}
	return id, nil
}

func (c *gatewayDeviceClient) GetByType(ctx context.Context, userID, deviceType string) (*ports.Device, error) {
	device := &ports.Device{}
	err := c.db.QueryRowContext(ctx,
		`SELECT id, user_id, device_type, device_name, token, is_connected, last_sync
		 FROM devices
		 WHERE user_id = $1 AND device_type = $2 AND is_connected = true
		 ORDER BY last_sync DESC
		 LIMIT 1`,
		userID, deviceType,
	).Scan(&device.ID, &device.UserID, &device.DeviceType, &device.DeviceName, &device.Token, &device.IsConnected, &device.LastSync)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, apperrors.Internal("failed to get device", err)
	}
	return device, nil
}

func (c *gatewayDeviceClient) UpdateLastSync(ctx context.Context, deviceID string) error {
	_, err := c.db.ExecContext(ctx, `UPDATE devices SET last_sync = NOW() WHERE id = $1`, deviceID)
	if err != nil {
		return apperrors.Internal("failed to update device last_sync", err)
	}
	return nil
}
