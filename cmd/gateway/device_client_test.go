package main

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGatewayDeviceClient(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	client := newGatewayDeviceClient(db)

	require.NotNil(t, client)
	assert.Same(t, db, client.db)
}

func TestGatewayDeviceClientCreateOrUpdateDevice(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectExec("INSERT INTO devices").
			WithArgs(sqlmock.AnyArg(), "user-1", "scale", "My Scale", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 1))

		id, err := newGatewayDeviceClient(db).CreateOrUpdateDevice(t.Context(), "user-1", "scale", "My Scale", `{"access_token":"tok"}`)

		require.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Contains(t, id, "dev_")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectExec("INSERT INTO devices").WillReturnError(assert.AnError)

		id, err := newGatewayDeviceClient(db).CreateOrUpdateDevice(t.Context(), "user-1", "scale", "My Scale", "{}")

		require.Error(t, err)
		assert.Empty(t, id)
		assert.Contains(t, err.Error(), "failed to create device")
	})
}

func TestGatewayDeviceClientGetByType(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		rows := sqlmock.NewRows([]string{"id", "user_id", "device_type", "device_name", "token", "is_connected", "last_sync"}).
			AddRow("dev-1", "user-1", "scale", "My Scale", `{"access_token":"tok"}`, true, nil)
		mock.ExpectQuery("SELECT id, user_id, device_type").
			WithArgs("user-1", "scale").
			WillReturnRows(rows)

		device, err := newGatewayDeviceClient(db).GetByType(t.Context(), "user-1", "scale")

		require.NoError(t, err)
		require.NotNil(t, device)
		assert.Equal(t, "dev-1", device.ID)
		assert.Equal(t, "user-1", device.UserID)
		assert.Equal(t, "scale", device.DeviceType)
		assert.Equal(t, "My Scale", device.DeviceName)
		assert.Equal(t, `{"access_token":"tok"}`, device.Token)
		assert.True(t, device.IsConnected)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no rows returns not found", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		empty := sqlmock.NewRows([]string{"id", "user_id", "device_type", "device_name", "token", "is_connected", "last_sync"})
		mock.ExpectQuery("SELECT id, user_id, device_type").WillReturnRows(empty)

		device, err := newGatewayDeviceClient(db).GetByType(t.Context(), "user-1", "scale")

		require.Error(t, err)
		assert.Nil(t, device)
		assert.ErrorIs(t, err, ErrDeviceNotFound)
	})

	t.Run("database error", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectQuery("SELECT id, user_id, device_type").WillReturnError(assert.AnError)

		device, err := newGatewayDeviceClient(db).GetByType(t.Context(), "user-1", "scale")

		require.Error(t, err)
		assert.Nil(t, device)
		assert.Contains(t, err.Error(), "failed to get device")
	})
}

func TestGatewayDeviceClientUpdateLastSync(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectExec("UPDATE devices SET last_sync").
			WithArgs("dev-1").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := newGatewayDeviceClient(db).UpdateLastSync(t.Context(), "dev-1")

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		db, mock := withingsNewDB(t)
		mock.ExpectExec("UPDATE devices SET last_sync").WillReturnError(assert.AnError)

		err := newGatewayDeviceClient(db).UpdateLastSync(t.Context(), "dev-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update device last_sync")
	})
}
