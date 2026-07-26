// Package validator provides input validation utilities for API requests.
package validator

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/training"
)

const (
	// MaxDurationWeeks is the maximum allowed plan duration in weeks.
	MaxDurationWeeks = 52
	// MaxAvailableDays is the maximum number of available training days per week.
	MaxAvailableDays = 7
)

var (
	ErrUserIDRequiredTraining = errors.New("user_id is required")
	ErrDurationWeeksRequired  = errors.New("duration_weeks must be greater than 0")
	ErrDurationWeeksTooLarge  = errors.New("duration_weeks must not exceed 52")
	ErrAvailableDaysRequired  = errors.New("available_days is required")
	ErrAvailableDaysTooMany   = errors.New("available_days must not exceed 7")
	ErrPlanIDRequired         = errors.New("plan_id is required")
	ErrWorkoutIDRequired      = errors.New("workout_id is required")
)

// ValidateGeneratePlanRequest validates a GeneratePlanRequest.
// DurationWeeks must be explicitly provided; defaults should be applied by the caller.
func ValidateGeneratePlanRequest(req *pb.GeneratePlanRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	if req.DurationWeeks <= 0 {
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

// ValidateCompleteWorkoutRequest validates a CompleteWorkoutRequest.
func ValidateCompleteWorkoutRequest(req *pb.CompleteWorkoutRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	if req.PlanId == "" {
		return status.Error(codes.InvalidArgument, ErrPlanIDRequired.Error())
	}
	if req.WorkoutId == "" {
		return status.Error(codes.InvalidArgument, ErrWorkoutIDRequired.Error())
	}
	return nil
}

// ValidateListPlansRequest validates a ListPlansRequest.
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

// ValidateGetProgressRequest validates a GetProgressRequest.
func ValidateGetProgressRequest(req *pb.GetProgressRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, ErrUserIDRequiredTraining.Error())
	}
	return nil
}
