package service

import (
	"context"
	"testing"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBiometricRepository struct {
	createFn         func(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	batchCreateFn    func(ctx context.Context, records []*entity.BiometricRecord) (int, error)
	getByUserIDFn    func(ctx context.Context, userID, metricType string, limit, offset int) ([]*entity.BiometricRecord, error)
	updateFn         func(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	deleteFn         func(ctx context.Context, id string) error
}

func (m *mockBiometricRepository) Create(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {
	if m.createFn != nil {
		return m.createFn(ctx, record)
	}
	return record, nil
}

func (m *mockBiometricRepository) BatchCreate(ctx context.Context, records []*entity.BiometricRecord) (int, error) {
	if m.batchCreateFn != nil {
		return m.batchCreateFn(ctx, records)
	}
	return len(records), nil
}

func (m *mockBiometricRepository) GetByUserID(ctx context.Context, userID, metricType string, limit, offset int) ([]*entity.BiometricRecord, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID, metricType, limit, offset)
	}
	return nil, nil
}

func (m *mockBiometricRepository) Update(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, record)
	}
	return record, nil
}

func (m *mockBiometricRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockBiometricRepository) GetLatest(ctx context.Context, userID, metricType string) (*entity.BiometricRecord, error) {
	return nil, nil
}

var _ port.BiometricRepository = (*mockBiometricRepository)(nil)

func TestAddRecord(t *testing.T) {
	now := time.Now()

	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{UserID: "", MetricType: "heart_rate", Value: 70, Timestamp: now}
		result, err := svc.AddRecord(context.Background(), record)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when metric_type is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{UserID: "user1", MetricType: "", Value: 70, Timestamp: now}
		result, err := svc.AddRecord(context.Background(), record)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("sets timestamp when zero", func(t *testing.T) {
		var captured *entity.BiometricRecord
		mock := &mockBiometricRepository{
			createFn: func(_ context.Context, r *entity.BiometricRecord) (*entity.BiometricRecord, error) {
				captured = r
				return r, nil
			},
		}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{UserID: "user1", MetricType: "heart_rate", Value: 70}
		_, err := svc.AddRecord(context.Background(), record)

		require.NoError(t, err)
		assert.False(t, captured.Timestamp.IsZero())
	})

	t.Run("passes valid record to repository", func(t *testing.T) {
		var captured *entity.BiometricRecord
		mock := &mockBiometricRepository{
			createFn: func(_ context.Context, r *entity.BiometricRecord) (*entity.BiometricRecord, error) {
				captured = r
				return r, nil
			},
		}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{UserID: "user1", MetricType: "heart_rate", Value: 70, Timestamp: now}
		result, err := svc.AddRecord(context.Background(), record)

		require.NoError(t, err)
		assert.Equal(t, record, result)
		assert.Equal(t, "user1", captured.UserID)
		assert.Equal(t, "heart_rate", captured.MetricType)
		assert.Equal(t, 70.0, captured.Value)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockBiometricRepository{
			createFn: func(_ context.Context, _ *entity.BiometricRecord) (*entity.BiometricRecord, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{UserID: "user1", MetricType: "heart_rate", Value: 70, Timestamp: now}
		result, err := svc.AddRecord(context.Background(), record)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}

func TestBatchAddRecords(t *testing.T) {
	now := time.Now()

	t.Run("returns validation error when empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		count, err := svc.BatchAddRecords(context.Background(), []*entity.BiometricRecord{})

		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when any record has empty user_id", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		records := []*entity.BiometricRecord{
			{UserID: "user1", MetricType: "hr", Value: 70, Timestamp: now},
			{UserID: "", MetricType: "hr", Value: 80, Timestamp: now},
		}

		count, err := svc.BatchAddRecords(context.Background(), records)

		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("sets timestamps for records with zero timestamps", func(t *testing.T) {
		var captured []*entity.BiometricRecord
		mock := &mockBiometricRepository{
			batchCreateFn: func(_ context.Context, r []*entity.BiometricRecord) (int, error) {
				captured = r
				return len(r), nil
			},
		}
		svc := NewBiometricService(mock)

		records := []*entity.BiometricRecord{
			{UserID: "user1", MetricType: "hr", Value: 70},
			{UserID: "user1", MetricType: "spo2", Value: 98},
		}

		_, err := svc.BatchAddRecords(context.Background(), records)

		require.NoError(t, err)
		for _, r := range captured {
			assert.False(t, r.Timestamp.IsZero())
		}
	})

	t.Run("delegates to repository with valid records", func(t *testing.T) {
		records := []*entity.BiometricRecord{
			{UserID: "user1", MetricType: "hr", Value: 70, Timestamp: now},
			{UserID: "user1", MetricType: "spo2", Value: 98, Timestamp: now},
		}

		mock := &mockBiometricRepository{
			batchCreateFn: func(_ context.Context, r []*entity.BiometricRecord) (int, error) {
				return len(r), nil
			},
		}
		svc := NewBiometricService(mock)

		count, err := svc.BatchAddRecords(context.Background(), records)

		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockBiometricRepository{
			batchCreateFn: func(_ context.Context, _ []*entity.BiometricRecord) (int, error) {
				return 0, apperrors.Internal("db error", nil)
			},
		}
		svc := NewBiometricService(mock)

		records := []*entity.BiometricRecord{
			{UserID: "user1", MetricType: "hr", Value: 70, Timestamp: now},
		}

		count, err := svc.BatchAddRecords(context.Background(), records)

		require.Error(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestGetRecords(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		records, err := svc.GetRecords(context.Background(), "", "hr", 10)

		require.Error(t, err)
		assert.Nil(t, records)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("defaults limit to 100 when <= 0", func(t *testing.T) {
		var capturedLimit int
		mock := &mockBiometricRepository{
			getByUserIDFn: func(_ context.Context, _ string, _ string, limit, _ int) ([]*entity.BiometricRecord, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		svc := NewBiometricService(mock)

		_, err := svc.GetRecords(context.Background(), "user1", "hr", 0)

		require.NoError(t, err)
		assert.Equal(t, 100, capturedLimit)
	})

	t.Run("caps limit at 10000", func(t *testing.T) {
		var capturedLimit int
		mock := &mockBiometricRepository{
			getByUserIDFn: func(_ context.Context, _ string, _ string, limit, _ int) ([]*entity.BiometricRecord, error) {
				capturedLimit = limit
				return nil, nil
			},
		}
		svc := NewBiometricService(mock)

		_, err := svc.GetRecords(context.Background(), "user1", "hr", 50000)

		require.NoError(t, err)
		assert.Equal(t, 10000, capturedLimit)
	})

	t.Run("passes valid params to repository", func(t *testing.T) {
		var capturedUserID, capturedMetricType string
		var capturedLimit int
		mock := &mockBiometricRepository{
			getByUserIDFn: func(_ context.Context, userID, metricType string, limit, _ int) ([]*entity.BiometricRecord, error) {
				capturedUserID = userID
				capturedMetricType = metricType
				capturedLimit = limit
				return []*entity.BiometricRecord{{ID: "r1"}}, nil
			},
		}
		svc := NewBiometricService(mock)

		records, err := svc.GetRecords(context.Background(), "user1", "hr", 10)

		require.NoError(t, err)
		assert.Equal(t, "user1", capturedUserID)
		assert.Equal(t, "hr", capturedMetricType)
		assert.Equal(t, 10, capturedLimit)
		require.Len(t, records, 1)
	})
}

func TestGetLatest(t *testing.T) {
	now := time.Now()

	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record, err := svc.GetLatest(context.Background(), "", "hr")

		require.Error(t, err)
		assert.Nil(t, record)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when metric_type is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record, err := svc.GetLatest(context.Background(), "user1", "")

		require.Error(t, err)
		assert.Nil(t, record)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns not found when repository returns empty results", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record, err := svc.GetLatest(context.Background(), "user1", "hr")

		require.Error(t, err)
		assert.Nil(t, record)
		assert.Equal(t, "NOT_FOUND", apperrors.Code(err))
	})

	t.Run("returns first record from repository", func(t *testing.T) {
		expected := &entity.BiometricRecord{ID: "r1", UserID: "user1", MetricType: "hr", Timestamp: now}
		mock := &mockBiometricRepository{
			getByUserIDFn: func(_ context.Context, _ string, _ string, _, _ int) ([]*entity.BiometricRecord, error) {
				return []*entity.BiometricRecord{expected}, nil
			},
		}
		svc := NewBiometricService(mock)

		record, err := svc.GetLatest(context.Background(), "user1", "hr")

		require.NoError(t, err)
		assert.Equal(t, expected, record)
	})

	t.Run("passes repository error through", func(t *testing.T) {
		mock := &mockBiometricRepository{
			getByUserIDFn: func(_ context.Context, _ string, _ string, _, _ int) ([]*entity.BiometricRecord, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewBiometricService(mock)

		record, err := svc.GetLatest(context.Background(), "user1", "hr")

		require.Error(t, err)
		assert.Nil(t, record)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}

func TestUpdateRecord(t *testing.T) {
	now := time.Now()

	t.Run("returns validation error when id is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{ID: "", UserID: "user1", MetricType: "hr", Value: 70, Timestamp: now}
		result, err := svc.UpdateRecord(context.Background(), record)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when value is negative", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{ID: "r1", UserID: "user1", MetricType: "hr", Value: -5, Timestamp: now}
		result, err := svc.UpdateRecord(context.Background(), record)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("delegates to repository create for valid record", func(t *testing.T) {
		var captured *entity.BiometricRecord
		mock := &mockBiometricRepository{
			createFn: func(_ context.Context, r *entity.BiometricRecord) (*entity.BiometricRecord, error) {
				captured = r
				return r, nil
			},
		}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{ID: "r1", UserID: "user1", MetricType: "hr", Value: 70, Timestamp: now}
		result, err := svc.UpdateRecord(context.Background(), record)

		require.NoError(t, err)
		assert.Equal(t, record, result)
		assert.Equal(t, "r1", captured.ID)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockBiometricRepository{
			createFn: func(_ context.Context, _ *entity.BiometricRecord) (*entity.BiometricRecord, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewBiometricService(mock)

		record := &entity.BiometricRecord{ID: "r1", UserID: "user1", MetricType: "hr", Value: 70, Timestamp: now}
		result, err := svc.UpdateRecord(context.Background(), record)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestDeleteRecord(t *testing.T) {
	t.Run("returns validation error when id is empty", func(t *testing.T) {
		mock := &mockBiometricRepository{}
		svc := NewBiometricService(mock)

		err := svc.DeleteRecord(context.Background(), "")

		require.Error(t, err)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("delegates to repository for valid id", func(t *testing.T) {
		var capturedID string
		mock := &mockBiometricRepository{
			deleteFn: func(_ context.Context, id string) error {
				capturedID = id
				return nil
			},
		}
		svc := NewBiometricService(mock)

		err := svc.DeleteRecord(context.Background(), "r1")

		require.NoError(t, err)
		assert.Equal(t, "r1", capturedID)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockBiometricRepository{
			deleteFn: func(_ context.Context, _ string) error {
				return apperrors.Internal("db error", nil)
			},
		}
		svc := NewBiometricService(mock)

		err := svc.DeleteRecord(context.Background(), "r1")

		require.Error(t, err)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}
