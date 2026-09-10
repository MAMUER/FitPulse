package pgx

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

func setupTrainingRepo(t *testing.T) (*TrainingRepositoryPGX, *mockDB) {
	t.Helper()
	mock := &mockDB{}
	repo := NewTrainingRepositoryPGX(mock)
	return repo, mock
}

func TestTrainingRepositoryPGX_CreatePlan_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()
	plan := &entity.TrainingPlan{
		ID:             "plan-1",
		UserID:         "user-1",
		Classification: "strength",
		DurationWeeks:  4,
		AvailableDays:  []int{1, 3, 5},
		PlanData:       map[string]interface{}{"weeks": 4},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*string)
			*ptr = "plan-1"
			return nil
		}}
	}

	result, err := repo.CreatePlan(ctx, plan)
	require.NoError(t, err)
	assert.Equal(t, "plan-1", result.ID)
}

func TestTrainingRepositoryPGX_CreatePlan_JSONMarshalError(t *testing.T) {
	repo, _ := setupTrainingRepo(t)
	ctx := context.Background()

	planData := map[string]interface{}{}
	planData["invalid"] = make(chan int)
	plan := &entity.TrainingPlan{
		PlanData: planData,
	}

	result, err := repo.CreatePlan(ctx, plan)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_CreatePlan_QueryRowError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	result, err := repo.CreatePlan(ctx, &entity.TrainingPlan{ID: "plan-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_GetPlan_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			if len(dest) >= 8 {
				if s, ok := dest[0].(*string); ok {
					*s = "plan-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "strength"
				}
				if i, ok := dest[3].(*int); ok {
					*i = 4
				}
				if p, ok := dest[4].(*[]int); ok {
					*p = []int{1, 3, 5}
				}
				if b, ok := dest[5].(*[]byte); ok {
					planData, _ := json.Marshal(map[string]interface{}{"weeks": 4})
					*b = planData
				}
				if t, ok := dest[6].(*time.Time); ok {
					*t = time.Now()
				}
				if t, ok := dest[7].(*time.Time); ok {
					*t = time.Now()
				}
			}
			return nil
		}}
	}

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")
	require.NoError(t, err)
	assert.Equal(t, "plan-1", result.ID)
	assert.Equal(t, "strength", result.Classification)
}

func TestTrainingRepositoryPGX_GetPlan_NotFound(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return pgx.ErrNoRows
		}}
	}

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestTrainingRepositoryPGX_GetPlan_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_GetPlan_UnmarshalError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			if len(dest) >= 8 {
				if s, ok := dest[0].(*string); ok {
					*s = "plan-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "strength"
				}
				if i, ok := dest[3].(*int); ok {
					*i = 4
				}
				if p, ok := dest[4].(*[]int); ok {
					*p = []int{1, 3, 5}
				}
				if b, ok := dest[5].(*[]byte); ok {
					*b = []byte("invalid json")
				}
				if t, ok := dest[6].(*time.Time); ok {
					*t = time.Now()
				}
				if t, ok := dest[7].(*time.Time); ok {
					*t = time.Now()
				}
			}
			return nil
		}}
	}

	result, err := repo.GetPlan(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_ListPlans_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()
	now := time.Now()
	planData := map[string]interface{}{"weeks": 4}
	planDataJSON, _ := json.Marshal(planData)

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		assert.Contains(t, query, "FROM training_plans")
		return newMockRows([][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, planDataJSON, now, now, 1},
		}, func(dest ...interface{}) error {
			if len(dest) >= 8 {
				if s, ok := dest[0].(*string); ok {
					*s = "plan-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "strength"
				}
				if i, ok := dest[3].(*int); ok {
					*i = 4
				}
				if p, ok := dest[4].(*[]int); ok {
					*p = []int{1, 3, 5}
				}
				if b, ok := dest[5].(*[]byte); ok {
					*b = planDataJSON
				}
				if t, ok := dest[6].(*time.Time); ok {
					*t = now
				}
				if t, ok := dest[7].(*time.Time); ok {
					*t = now
				}
				if t, ok := dest[8].(*int); ok {
					*t = 1
				}
			}
			return nil
		}), nil
	}

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "plan-1", result[0].ID)
	assert.Equal(t, 1, total)
}

func TestTrainingRepositoryPGX_ListPlans_Empty(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return newMockRows([][]interface{}{}, func(dest ...interface{}) error {
		return nil
	}), nil
	}

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
}

func TestTrainingRepositoryPGX_ListPlans_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return nil, assert.AnError
	}

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_ListPlans_UnmarshalError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return newMockRows([][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte("invalid json"), time.Now(), time.Now()},
		}, func(dest ...interface{}) error {
			if len(dest) >= 8 {
				if s, ok := dest[0].(*string); ok {
					*s = "plan-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "strength"
				}
				if i, ok := dest[3].(*int); ok {
					*i = 4
				}
				if p, ok := dest[4].(*[]int); ok {
					*p = []int{1, 3, 5}
				}
				if b, ok := dest[5].(*[]byte); ok {
					*b = []byte("invalid json")
				}
				if t, ok := dest[6].(*time.Time); ok {
					*t = time.Now()
				}
				if t, ok := dest[7].(*time.Time); ok {
					*t = time.Now()
				}
			}
			return nil
		}), nil
	}

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*int)
			*ptr = 1
			return nil
		}}
	}

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_ListPlans_CountError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return newMockRows([][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte{}, time.Now(), time.Now()},
		}, func(dest ...interface{}) error {
			if len(dest) >= 8 {
				if s, ok := dest[0].(*string); ok {
					*s = "plan-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "strength"
				}
				if i, ok := dest[3].(*int); ok {
					*i = 4
				}
				if p, ok := dest[4].(*[]int); ok {
					*p = []int{1, 3, 5}
				}
				if b, ok := dest[5].(*[]byte); ok {
					*b = []byte{}
				}
				if t, ok := dest[6].(*time.Time); ok {
					*t = time.Now()
				}
				if t, ok := dest[7].(*time.Time); ok {
					*t = time.Now()
				}
				if len(dest) > 8 {
					return assert.AnError
				}
			}
			return nil
		}), nil
	}

	result, total, err := repo.ListPlans(ctx, "user-1", 1, 10)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_CompleteWorkout_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		assert.Contains(t, query, "UPDATE training_plans")
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	err := repo.CompleteWorkout(ctx, "user-1", "plan-1")
	require.NoError(t, err)
}

func TestTrainingRepositoryPGX_CompleteWorkout_Error(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, assert.AnError
	}

	err := repo.CompleteWorkout(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_GetProgress_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*int)
			*ptr = 10
			ptr = dest[1].(*int)
			*ptr = 5
			return nil
		}}
	}

	result, err := repo.GetProgress(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 10, result["total_plans"])
	assert.Equal(t, 5, result["completed_workouts"])
	assert.Equal(t, 50.0, result["completion_rate"])
}

func TestTrainingRepositoryPGX_GetProgress_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	result, err := repo.GetProgress(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_GetProgress_PanicOnZeroPlans(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*int)
			*ptr = 0
			ptr = dest[1].(*int)
			*ptr = 0
			return nil
		}}
	}

	result, err := repo.GetProgress(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 0, result["total_plans"])
	assert.Equal(t, 0, result["completed_workouts"])
	rate, ok := result["completion_rate"].(float64)
	require.True(t, ok)
	assert.True(t, math.IsNaN(rate))
}

func TestTrainingRepositoryPGX_GetAchievements_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()
	now := time.Now()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		assert.Contains(t, query, "SELECT id")
		return newMockRows([][]interface{}{
			{"ach-1", "user-1", "strength", "First Lift", "Lift 100kg", now},
		}, nil), nil
	}

	result, err := repo.GetAchievements(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "ach-1", result[0].ID)
	assert.Equal(t, "First Lift", result[0].Title)
}

func TestTrainingRepositoryPGX_GetAchievements_Empty(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return newMockRows([][]interface{}{}, func(dest ...interface{}) error {
			return nil
		}), nil
	}

	result, err := repo.GetAchievements(ctx, "user-1")
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestTrainingRepositoryPGX_GetAchievements_QueryError(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryFunc = func(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
		return nil, assert.AnError
	}

	result, err := repo.GetAchievements(ctx, "user-1")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_CreateAchievement_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			ptr := dest[0].(*string)
			*ptr = "ach-1"
			return nil
		}}
	}

	achievement := &entity.Achievement{
		UserID:      "user-1",
		Type:        "strength",
		Title:       "First Lift",
		Description: "Lift 100kg",
		EarnedAt:    time.Now(),
	}
	result, err := repo.CreateAchievement(ctx, achievement)
	require.NoError(t, err)
	assert.Equal(t, "ach-1", result.ID)
}

func TestTrainingRepositoryPGX_CreateAchievement_Error(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	result, err := repo.CreateAchievement(ctx, &entity.Achievement{UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_DeletePlan_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		assert.Contains(t, query, "DELETE FROM training_plans")
		return pgconn.NewCommandTag("DELETE 1"), nil
	}

	err := repo.DeletePlan(ctx, "user-1", "plan-1")
	require.NoError(t, err)
}

func TestTrainingRepositoryPGX_DeletePlan_Error(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.execFunc = func(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
		return pgconn.CommandTag{}, assert.AnError
	}

	err := repo.DeletePlan(ctx, "user-1", "plan-1")
	require.Error(t, err)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}

func TestTrainingRepositoryPGX_UpdatePlan_Success(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()
	now := time.Now()
	plan := &entity.TrainingPlan{
		ID:             "plan-1",
		UserID:         "user-1",
		Classification: "cardio",
		DurationWeeks:  8,
		AvailableDays:  []int{1, 2, 3, 4, 5},
		CreatedAt:      now,
	}

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			if len(dest) >= 7 {
				if s, ok := dest[0].(*string); ok {
					*s = "plan-1"
				}
				if s, ok := dest[1].(*string); ok {
					*s = "user-1"
				}
				if s, ok := dest[2].(*string); ok {
					*s = "cardio"
				}
				if i, ok := dest[3].(*int); ok {
					*i = 8
				}
				if p, ok := dest[4].(*[]int); ok {
					*p = []int{1, 2, 3, 4, 5}
				}
				if t, ok := dest[5].(*time.Time); ok {
					*t = now
				}
				if t, ok := dest[6].(*time.Time); ok {
					*t = now
				}
			}
			return nil
		}}
	}

	result, err := repo.UpdatePlan(ctx, plan)
	require.NoError(t, err)
	assert.Equal(t, "plan-1", result.ID)
	assert.Equal(t, "cardio", result.Classification)
}

func TestTrainingRepositoryPGX_UpdatePlan_NotFound(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return pgx.ErrNoRows
		}}
	}

	result, err := repo.UpdatePlan(ctx, &entity.TrainingPlan{ID: "plan-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.IsNotFound(err))
}

func TestTrainingRepositoryPGX_UpdatePlan_Error(t *testing.T) {
	repo, mock := setupTrainingRepo(t)
	ctx := context.Background()

	mock.queryRowFunc = func(ctx context.Context, query string, args ...interface{}) pgx.Row {
		return &mockRow{scanFunc: func(dest ...interface{}) error {
			return assert.AnError
		}}
	}

	result, err := repo.UpdatePlan(ctx, &entity.TrainingPlan{ID: "plan-1", UserID: "user-1"})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, apperrors.Code(err) == "INTERNAL")
}
