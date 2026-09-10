package pgx_test

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/repository/pgx"
	"github.com/MAMUER/project/internal/testcontainers"
)

func isDockerAvailable() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	return err == nil
}

func setupBiometricRepo(t *testing.T) (*pgx.BiometricRepositoryPGX, func()) {
	t.Helper()
	ctx := context.Background()
	ctr := testcontainers.StartInfrastructure(t)

	connStr := fmt.Sprintf("postgres://testuser:testpass@%s:%d/testdb?sslmode=disable",
		ctr.PostgresHost, ctr.PostgresPort)
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	repo := pgx.NewBiometricRepositoryPGX(pool)
	cleanup := func() {
		pool.Close()
	}

	return repo, cleanup
}

func TestBiometricRepositoryPGX_CreateAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	repo, cleanup := setupBiometricRepo(t)
	defer cleanup()

	ctx := context.Background()
	record := &entity.BiometricRecord{
		ID:         "test-bio-1",
		UserID:     "user-1",
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Now(),
		DeviceType: "apple_watch",
		Source:     "test",
	}

	created, err := repo.Create(ctx, record)
	require.NoError(t, err)
	assert.Equal(t, "test-bio-1", created.ID)

	got, err := repo.GetByID(ctx, "test-bio-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", got.UserID)
	assert.Equal(t, "heart_rate", got.MetricType)
}

func TestBiometricRepositoryPGX_GetLatest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	repo, cleanup := setupBiometricRepo(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	for i := 0; i < 3; i++ {
		_, err := repo.Create(ctx, &entity.BiometricRecord{
			ID:         "test-bio-latest-" + string(rune('a'+i)),
			UserID:     "user-1",
			MetricType: "heart_rate",
			Value:      float64(70 + i),
			Timestamp:  now.Add(time.Duration(i) * time.Minute),
			DeviceType: "apple_watch",
			Source:     "test",
		})
		require.NoError(t, err)
	}

	latest, err := repo.GetLatest(ctx, "user-1", "heart_rate")
	require.NoError(t, err)
	assert.Equal(t, float64(72), latest.Value)
}

func TestBiometricRepositoryPGX_Update_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	repo, cleanup := setupBiometricRepo(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.Update(ctx, &entity.BiometricRecord{
		ID:     "missing",
		UserID: "user-1",
	})
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestBiometricRepositoryPGX_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	repo, cleanup := setupBiometricRepo(t)
	defer cleanup()

	ctx := context.Background()
	_, err := repo.Create(ctx, &entity.BiometricRecord{
		ID:         "test-bio-del",
		UserID:     "user-1",
		MetricType: "heart_rate",
		Value:      80,
		Timestamp:  time.Now(),
		DeviceType: "apple_watch",
		Source:     "test",
	})
	require.NoError(t, err)

	err = repo.Delete(ctx, "test-bio-del")
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, "test-bio-del")
	require.Error(t, err)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestBiometricRepositoryPGX_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping integration test")
	}

	repo, cleanup := setupBiometricRepo(t)
	defer cleanup()

	ctx := context.Background()
	records := []*entity.BiometricRecord{
		{UserID: "user-1", MetricType: "heart_rate", Value: 70, Timestamp: time.Now(), DeviceType: "apple_watch", Source: "test"},
		{UserID: "user-1", MetricType: "heart_rate", Value: 71, Timestamp: time.Now(), DeviceType: "apple_watch", Source: "test"},
	}

	inserted, err := repo.BatchCreate(ctx, records)
	require.NoError(t, err)
	assert.Equal(t, 2, inserted)
}
