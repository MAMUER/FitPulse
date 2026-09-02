package service

import (
	"context"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type trainingService struct {
	training port.TrainingRepository
}

func NewTrainingService(training port.TrainingRepository) TrainingService {
	return &trainingService{training: training}
}

func (s *trainingService) GeneratePlan(ctx context.Context, userID, classification string, durationWeeks int, availableDays []int) (*entity.TrainingPlan, error) {
	if userID == "" {
		return nil, apperrors.Validation("user_id is required")
	}
	if classification == "" {
		return nil, apperrors.Validation("classification is required")
	}
	if durationWeeks <= 0 {
		durationWeeks = 4
	}
	if len(availableDays) == 0 {
		availableDays = []int{1, 3, 5}
	}

	plan := &entity.TrainingPlan{
		ID:             generateID(),
		UserID:         userID,
		Classification: classification,
		DurationWeeks:  durationWeeks,
		AvailableDays:  availableDays,
		PlanData:       map[string]interface{}{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return s.training.CreatePlan(ctx, plan)
}

func (s *trainingService) GetPlan(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error) {
	if userID == "" || planID == "" {
		return nil, apperrors.Validation("user_id and plan_id are required")
	}
	return s.training.GetPlan(ctx, userID, planID)
}

func (s *trainingService) ListPlans(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error) {
	if userID == "" {
		return nil, 0, apperrors.Validation("user_id is required")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.training.ListPlans(ctx, userID, page, pageSize)
}

func (s *trainingService) CompleteWorkout(ctx context.Context, userID, planID string, rating int, feedback string) error {
	if userID == "" || planID == "" {
		return apperrors.Validation("user_id and plan_id are required")
	}
	return s.training.CompleteWorkout(ctx, userID, planID)
}

func (s *trainingService) GetProgress(ctx context.Context, userID string) (map[string]interface{}, error) {
	if userID == "" {
		return nil, apperrors.Validation("user_id is required")
	}
	return s.training.GetProgress(ctx, userID)
}

func (s *trainingService) GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	if userID == "" {
		return nil, apperrors.Validation("user_id is required")
	}
	return s.training.GetAchievements(ctx, userID)
}
