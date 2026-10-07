package ports

import "context"

type DeviceClient interface {
	CreateOrUpdateDevice(ctx context.Context, userID, deviceType, deviceName, token string) (string, error)
	GetByType(ctx context.Context, userID, deviceType string) (*Device, error)
	UpdateLastSync(ctx context.Context, deviceID string) error
}

type Device struct {
	ID          string
	UserID      string
	DeviceType  string
	DeviceName  string
	Token       string
	IsConnected bool
	LastSync    interface{}
}
