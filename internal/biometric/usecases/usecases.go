// Package usecases provides application use cases for biometric data processing.
package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/MAMUER/project/internal/biometric/adapters"
	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/biometric/medical"
)

// FetchBiometricDataRequest represents a request to fetch biometric data.
type FetchBiometricDataRequest struct {
	UserID      string
	MetricTypes []string
	Sources     []domain.BiometricSource
}

// FetchBiometricDataResponse represents the response from fetching biometric data.
type FetchBiometricDataResponse struct {
	Samples   []domain.BiometricSample
	Errors    []string
	Sources   int
	Available int
}

// FetchBiometricDataUseCase fetches and aggregates biometric data from multiple sources.
type FetchBiometricDataUseCase struct {
	aggregator domain.BiometricSource
}

// NewFetchBiometricDataUseCase creates a new use case for fetching biometric data.
func NewFetchBiometricDataUseCase(aggregator domain.BiometricSource) *FetchBiometricDataUseCase {
	return &FetchBiometricDataUseCase{aggregator: aggregator}
}

// Execute fetches biometric data from the configured sources.
func (uc *FetchBiometricDataUseCase) Execute(ctx context.Context, req FetchBiometricDataRequest) (*FetchBiometricDataResponse, error) {
	if len(req.Sources) == 0 {
		return nil, errors.New("no sources configured")
	}

	composite := adapters.NewCompositeBiometricSource(req.Sources...)
	samples, err := composite.Fetch(ctx, req.UserID, req.MetricTypes)
	if err != nil {
		return nil, fmt.Errorf("fetch biometric data: %w", err)
	}

	response := &FetchBiometricDataResponse{
		Samples:   samples,
		Sources:   len(req.Sources),
		Available: len(req.Sources),
	}

	return response, nil
}

// ValidateTrainingPlanUseCase validates training plans against medical constraints.
type ValidateTrainingPlanUseCase struct {
	medicalService *medical.MedicalService
}

// NewValidateTrainingPlanUseCase creates a new use case for validating training plans.
func NewValidateTrainingPlanUseCase(medicalService *medical.MedicalService) *ValidateTrainingPlanUseCase {
	return &ValidateTrainingPlanUseCase{medicalService: medicalService}
}

// ValidateTrainingPlanRequest represents a request to validate a training plan.
type ValidateTrainingPlanRequest struct {
	UserID  string
	Workout *domain.WorkoutPlan
}

// ValidateTrainingPlanResponse represents the response from validation.
type ValidateTrainingPlanResponse struct {
	Violations     []domain.ConstraintViolation
	RequiresReview bool
	Allowed        bool
}

// Execute validates a training plan against medical constraints.
func (uc *ValidateTrainingPlanUseCase) Execute(ctx context.Context, req ValidateTrainingPlanRequest) (*ValidateTrainingPlanResponse, error) {
	if req.Workout == nil {
		return nil, errors.New("workout plan is required")
	}

	violations, requiresReview := uc.medicalService.EvaluateWorkout(ctx, req.UserID, req.Workout)

	allowed := true
	for _, v := range violations {
		if v.Action == "avoid" {
			allowed = false
			break
		}
	}

	return &ValidateTrainingPlanResponse{
		Violations:     violations,
		RequiresReview: requiresReview,
		Allowed:        allowed,
	}, nil
}

// ImportDeviceDataUseCase imports biometric data from external devices.
type ImportDeviceDataUseCase struct {
	source domain.BiometricSource
	sink   domain.BiometricSink
}

// NewImportDeviceDataUseCase creates a new use case for importing device data.
func NewImportDeviceDataUseCase(source domain.BiometricSource, sink domain.BiometricSink) *ImportDeviceDataUseCase {
	return &ImportDeviceDataUseCase{
		source: source,
		sink:   sink,
	}
}

// ImportDeviceDataRequest represents a request to import device data.
type ImportDeviceDataRequest struct {
	UserID      string
	DeviceID    string
	MetricTypes []string
}

// ImportDeviceDataResponse represents the response from importing device data.
type ImportDeviceDataResponse struct {
	Imported int
	Errors   []string
}

// Execute imports biometric data from a device.
func (uc *ImportDeviceDataUseCase) Execute(ctx context.Context, req ImportDeviceDataRequest) (*ImportDeviceDataResponse, error) {
	if req.DeviceID == "" {
		return nil, errors.New("device ID is required")
	}

	samples, err := uc.source.Fetch(ctx, req.UserID, req.MetricTypes)
	if err != nil {
		return nil, fmt.Errorf("fetch device data: %w", err)
	}

	if len(samples) == 0 {
		return &ImportDeviceDataResponse{
			Imported: 0,
			Errors:   []string{"no data available from device"},
		}, nil
	}

	if err := uc.sink.Store(ctx, samples); err != nil {
		return nil, fmt.Errorf("store device data: %w", err)
	}

	return &ImportDeviceDataResponse{
		Imported: len(samples),
		Errors:   nil,
	}, nil
}
