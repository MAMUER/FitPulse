package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type mockTrainingRepository struct {
	createPlanFn      func(ctx context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error)
	getPlanFn         func(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error)
	listPlansFn       func(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error)
	completeWorkoutFn func(ctx context.Context, userID, planID string) error
	getProgressFn     func(ctx context.Context, userID string) (map[string]interface{}, error)
	getAchievementsFn func(ctx context.Context, userID string) ([]*entity.Achievement, error)
}

func (m *mockTrainingRepository) CreatePlan(ctx context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
	if m.createPlanFn != nil {
		return m.createPlanFn(ctx, plan)
	}
	return plan, nil
}

func (m *mockTrainingRepository) GetPlan(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error) {
	if m.getPlanFn != nil {
		return m.getPlanFn(ctx, userID, planID)
	}
	return &entity.TrainingPlan{ID: planID, UserID: userID}, nil
}

func (m *mockTrainingRepository) ListPlans(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error) {
	if m.listPlansFn != nil {
		return m.listPlansFn(ctx, userID, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockTrainingRepository) CompleteWorkout(ctx context.Context, userID, planID string) error {
	if m.completeWorkoutFn != nil {
		return m.completeWorkoutFn(ctx, userID, planID)
	}
	return nil
}

func (m *mockTrainingRepository) GetProgress(ctx context.Context, userID string) (map[string]interface{}, error) {
	if m.getProgressFn != nil {
		return m.getProgressFn(ctx, userID)
	}
	return map[string]interface{}{}, nil
}

func (m *mockTrainingRepository) GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	if m.getAchievementsFn != nil {
		return m.getAchievementsFn(ctx, userID)
	}
	return nil, nil
}

var _ port.TrainingRepository = (*mockTrainingRepository)(nil)

func TestGeneratePlan(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		plan, err := svc.GeneratePlan(context.Background(), "", "beginner", 4, []int{1, 3, 5})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when classification is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		plan, err := svc.GeneratePlan(context.Background(), "user1", "", 4, []int{1, 3, 5})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("defaults duration to 4 when <= 0", func(t *testing.T) {
		var capturedDuration int
		mock := &mockTrainingRepository{
			createPlanFn: func(_ context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
				capturedDuration = plan.DurationWeeks
				return plan, nil
			},
		}
		svc := NewTrainingService(mock)

		_, err := svc.GeneratePlan(context.Background(), "user1", "beginner", 0, []int{1, 3, 5})

		require.NoError(t, err)
		assert.Equal(t, 4, capturedDuration)
	})

	t.Run("defaults available days when empty", func(t *testing.T) {
		var capturedDays []int
		mock := &mockTrainingRepository{
			createPlanFn: func(_ context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
				capturedDays = plan.AvailableDays
				return plan, nil
			},
		}
		svc := NewTrainingService(mock)

		_, err := svc.GeneratePlan(context.Background(), "user1", "beginner", 4, []int{})

		require.NoError(t, err)
		assert.Equal(t, []int{1, 3, 5}, capturedDays)
	})

	t.Run("creates plan with correct fields", func(t *testing.T) {
		var captured *entity.TrainingPlan
		mock := &mockTrainingRepository{
			createPlanFn: func(_ context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error) {
				captured = plan
				return plan, nil
			},
		}
		svc := NewTrainingService(mock)

		plan, err := svc.GeneratePlan(context.Background(), "user1", "beginner", 8, []int{2, 4, 6})

		require.NoError(t, err)
		assert.Equal(t, plan, captured)
		assert.Equal(t, "user1", captured.UserID)
		assert.Equal(t, "beginner", captured.Classification)
		assert.Equal(t, 8, captured.DurationWeeks)
		assert.Equal(t, []int{2, 4, 6}, captured.AvailableDays)
		assert.NotNil(t, captured.PlanData)
		assert.False(t, captured.CreatedAt.IsZero())
		assert.False(t, captured.UpdatedAt.IsZero())
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockTrainingRepository{
			createPlanFn: func(_ context.Context, _ *entity.TrainingPlan) (*entity.TrainingPlan, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewTrainingService(mock)

		plan, err := svc.GeneratePlan(context.Background(), "user1", "beginner", 4, []int{1, 3, 5})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}

func TestGetPlan(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		plan, err := svc.GetPlan(context.Background(), "", "plan1")

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when plan_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		plan, err := svc.GetPlan(context.Background(), "user1", "")

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("delegates to repository with valid ids", func(t *testing.T) {
		expected := &entity.TrainingPlan{ID: "plan1", UserID: "user1", Classification: "beginner"}
		mock := &mockTrainingRepository{
			getPlanFn: func(_ context.Context, userID, planID string) (*entity.TrainingPlan, error) {
				if userID == "user1" && planID == "plan1" {
					return expected, nil
				}
				return nil, apperrors.NotFound("plan not found")
			},
		}
		svc := NewTrainingService(mock)

		plan, err := svc.GetPlan(context.Background(), "user1", "plan1")

		require.NoError(t, err)
		assert.Equal(t, expected, plan)
	})

	t.Run("returns repository not found error", func(t *testing.T) {
		mock := &mockTrainingRepository{
			getPlanFn: func(_ context.Context, _ string, _ string) (*entity.TrainingPlan, error) {
				return nil, apperrors.NotFound("plan not found")
			},
		}
		svc := NewTrainingService(mock)

		plan, err := svc.GetPlan(context.Background(), "user1", "plan1")

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.Equal(t, "NOT_FOUND", apperrors.Code(err))
	})
}

func TestListPlans(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		plans, total, err := svc.ListPlans(context.Background(), "", 1, 20)

		require.Error(t, err)
		assert.Nil(t, plans)
		assert.Equal(t, 0, total)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("defaults page to 1 when <= 0", func(t *testing.T) {
		var capturedPage int
		mock := &mockTrainingRepository{
			listPlansFn: func(_ context.Context, _ string, page, _ int) ([]*entity.TrainingPlan, int, error) {
				capturedPage = page
				return nil, 0, nil
			},
		}
		svc := NewTrainingService(mock)

		_, _, err := svc.ListPlans(context.Background(), "user1", 0, 10)

		require.NoError(t, err)
		assert.Equal(t, 1, capturedPage)
	})

	t.Run("defaults pageSize to 20 when <= 0", func(t *testing.T) {
		var capturedPageSize int
		mock := &mockTrainingRepository{
			listPlansFn: func(_ context.Context, _ string, _, pageSize int) ([]*entity.TrainingPlan, int, error) {
				capturedPageSize = pageSize
				return nil, 0, nil
			},
		}
		svc := NewTrainingService(mock)

		_, _, err := svc.ListPlans(context.Background(), "user1", 1, 0)

		require.NoError(t, err)
		assert.Equal(t, 20, capturedPageSize)
	})

	t.Run("delegates to repository with valid params", func(t *testing.T) {
		plans := []*entity.TrainingPlan{{ID: "p1", UserID: "user1"}}
		mock := &mockTrainingRepository{
			listPlansFn: func(_ context.Context, _ string, page, pageSize int) ([]*entity.TrainingPlan, int, error) {
				return plans, 1, nil
			},
		}
		svc := NewTrainingService(mock)

		result, total, err := svc.ListPlans(context.Background(), "user1", 2, 10)

		require.NoError(t, err)
		assert.Equal(t, plans, result)
		assert.Equal(t, 1, total)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockTrainingRepository{
			listPlansFn: func(_ context.Context, _ string, _, _ int) ([]*entity.TrainingPlan, int, error) {
				return nil, 0, apperrors.Internal("db error", nil)
			},
		}
		svc := NewTrainingService(mock)

		plans, total, err := svc.ListPlans(context.Background(), "user1", 1, 20)

		require.Error(t, err)
		assert.Nil(t, plans)
		assert.Equal(t, 0, total)
	})
}

func TestCompleteWorkout(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		err := svc.CompleteWorkout(context.Background(), "", "plan1", 5, "great")

		require.Error(t, err)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when plan_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		err := svc.CompleteWorkout(context.Background(), "user1", "", 5, "great")

		require.Error(t, err)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("delegates to repository", func(t *testing.T) {
		var capturedUserID, capturedPlanID string
		mock := &mockTrainingRepository{
			completeWorkoutFn: func(_ context.Context, userID, planID string) error {
				capturedUserID = userID
				capturedPlanID = planID
				return nil
			},
		}
		svc := NewTrainingService(mock)

		err := svc.CompleteWorkout(context.Background(), "user1", "plan1", 5, "great")

		require.NoError(t, err)
		assert.Equal(t, "user1", capturedUserID)
		assert.Equal(t, "plan1", capturedPlanID)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockTrainingRepository{
			completeWorkoutFn: func(_ context.Context, _ string, _ string) error {
				return apperrors.Internal("db error", nil)
			},
		}
		svc := NewTrainingService(mock)

		err := svc.CompleteWorkout(context.Background(), "user1", "plan1", 5, "great")

		require.Error(t, err)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}

func TestGetProgress(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		progress, err := svc.GetProgress(context.Background(), "")

		require.Error(t, err)
		assert.Nil(t, progress)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns progress from repository", func(t *testing.T) {
		expected := map[string]interface{}{"completed": 5, "total": 10}
		mock := &mockTrainingRepository{
			getProgressFn: func(_ context.Context, _ string) (map[string]interface{}, error) {
				return expected, nil
			},
		}
		svc := NewTrainingService(mock)

		progress, err := svc.GetProgress(context.Background(), "user1")

		require.NoError(t, err)
		assert.Equal(t, expected, progress)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockTrainingRepository{
			getProgressFn: func(_ context.Context, _ string) (map[string]interface{}, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewTrainingService(mock)

		progress, err := svc.GetProgress(context.Background(), "user1")

		require.Error(t, err)
		assert.Nil(t, progress)
	})
}

func TestGetAchievements(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mock := &mockTrainingRepository{}
		svc := NewTrainingService(mock)

		achievements, err := svc.GetAchievements(context.Background(), "")

		require.Error(t, err)
		assert.Nil(t, achievements)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns achievements from repository", func(t *testing.T) {
		expected := []*entity.Achievement{{ID: "a1", Title: "First Workout"}}
		mock := &mockTrainingRepository{
			getAchievementsFn: func(_ context.Context, _ string) ([]*entity.Achievement, error) {
				return expected, nil
			},
		}
		svc := NewTrainingService(mock)

		achievements, err := svc.GetAchievements(context.Background(), "user1")

		require.NoError(t, err)
		assert.Equal(t, expected, achievements)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mock := &mockTrainingRepository{
			getAchievementsFn: func(_ context.Context, _ string) ([]*entity.Achievement, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewTrainingService(mock)

		achievements, err := svc.GetAchievements(context.Background(), "user1")

		require.Error(t, err)
		assert.Nil(t, achievements)
	})
}
