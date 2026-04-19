// internal/validator/training.go
package validator

import (
	"errors"

	pb "github.com/MAMUER/Project/api/gen/training"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	MaxDurationWeeks = 52
	MaxAvailableDays = 7
)

var (
	ErrUserIDRequiredTraining = errors.New("user_id is required")
	ErrDurationWeeksRequired  = errors.New("duration_weeks must be greater than 0")
	ErrDurationWeeksTooLarge  = errors.New("duration_weeks must not exceed 52")
	ErrAvailableDaysRequired  = errors.New("available_days is required")
	ErrAvailableDaysTooMany   = errors.New("available_days must not exceed 7")
	ErrPlanIdRequired         = errors.New("plan_id is required")
	ErrWorkoutIdRequired      = errors.New("workout_id is required")
)

// ValidateGeneratePlanRequest checks request for training plan generation.
// DurationWeeks is optional - defaults to 4 if not specified.
func ValidateGeneratePlanRequest(req *pb.GeneratePlanRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	// Default duration to 4 weeks if not specified (for ML-generated plans)
	if req.DurationWeeks == 0 {
		req.DurationWeeks = 4
	}
	if req.DurationWeeks < 0 {
		return status.Error(codes.InvalidArgument, ErrDurationWeeksRequired.Error())
	}
	if req.DurationWeeks > MaxDurationWeeks {
		return status.Error(codes.InvalidArgument, ErrDurationWeeksTooLarge.Error())
	}
	if len(req.AvailableDays) == 0 {
		return status.Error(codes.InvalidArgument, ErrAvailableDaysRequired.Error())
	}
	if len(req.AvailableDays) > MaxAvailableDays {
		return status.Error(codes.InvalidArgument, ErrAvailableDaysTooMany.Error())
	}
	return nil
}

// ValidateCompleteWorkoutRequest проверяет запрос завершения тренировки
func ValidateCompleteWorkoutRequest(req *pb.CompleteWorkoutRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	if req.PlanId == "" {
		return status.Error(codes.InvalidArgument, ErrPlanIdRequired.Error())
	}
	if req.WorkoutId == "" {
		return status.Error(codes.InvalidArgument, ErrWorkoutIdRequired.Error())
	}
	return nil
}

// ValidateListPlansRequest проверяет запрос списка планов
func ValidateListPlansRequest(req *pb.ListPlansRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	if req.PageSize <= 0 {
		return status.Error(codes.InvalidArgument, "page_size must be greater than 0")
	}
	if req.Page < 0 {
		return status.Error(codes.InvalidArgument, "page must be non-negative")
	}
	return nil
}

// ValidateGetProgressRequest проверяет запрос прогресса
func ValidateGetProgressRequest(req *pb.GetProgressRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	return nil
}
