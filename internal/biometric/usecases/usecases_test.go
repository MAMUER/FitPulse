package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/biometric/adapters"
	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/biometric/medical"
)

// mockMedicalService is a mock implementation of medical.MedicalRepository
type mockMedicalService struct {
	mock.Mock
}

func (m *mockMedicalService) GetActiveConstraints(ctx context.Context, userID string) ([]domain.MedicalConstraint, error) {
	args := m.Called(ctx, userID)
	violations, _ := args.Get(0).([]domain.MedicalConstraint)
	err := args.Error(1)
	return violations, err
}

func (m *mockMedicalService) SaveConstraint(ctx context.Context, constraint domain.MedicalConstraint) error {
	args := m.Called(ctx, constraint)
	return args.Error(0)
}

func (m *mockMedicalService) DeleteConstraint(ctx context.Context, constraintID string) error {
	args := m.Called(ctx, constraintID)
	return args.Error(0)
}

func (m *mockMedicalService) GetConstraintByCode(ctx context.Context, code string) ([]domain.MedicalConstraint, error) {
	args := m.Called(ctx, code)
	constraints, _ := args.Get(0).([]domain.MedicalConstraint)
	err := args.Error(1)
	return constraints, err
}

// mockBiometricSource is a mock implementation of BiometricSource
type mockBiometricSource struct {
	mock.Mock
	deviceType string
}

func newMockBiometricSource() *mockBiometricSource {
	return &mockBiometricSource{deviceType: "test"}
}

func (m *mockBiometricSource) Fetch(ctx context.Context, userID string, metricTypes []string) ([]domain.BiometricSample, error) {
	args := m.Called(ctx, userID, metricTypes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.BiometricSample), args.Error(1)
}

func (m *mockBiometricSource) Supports(metricType string) bool {
	args := m.Called(metricType)
	return args.Bool(0)
}

func (m *mockBiometricSource) DeviceType() string {
	return m.deviceType
}

func (m *mockBiometricSource) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// mockBiometricSink is a mock implementation of BiometricSink
type mockBiometricSink struct {
	mock.Mock
}

func (m *mockBiometricSink) Store(ctx context.Context, samples []domain.BiometricSample) error {
	args := m.Called(ctx, samples)
	return args.Error(0)
}

func (m *mockBiometricSink) BatchStore(ctx context.Context, userID string, samples []domain.BiometricSample) error {
	args := m.Called(ctx, userID, samples)
	return args.Error(0)
}

func TestNewFetchBiometricDataUseCase(t *testing.T) {
	source := newMockBiometricSource()
	uc := NewFetchBiometricDataUseCase(source)
	assert.NotNil(t, uc)
	assert.NotNil(t, uc.aggregator)
}

func TestFetchBiometricDataUseCase_Execute_NoSources(t *testing.T) {
	uc := NewFetchBiometricDataUseCase(nil)
	req := FetchBiometricDataRequest{
		UserID:      "user-1",
		MetricTypes: []string{"heart_rate"},
		Sources:     []domain.BiometricSource{},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "no sources configured", err.Error())
}

func TestFetchBiometricDataUseCase_Execute_Success(t *testing.T) {
	source := newMockBiometricSource()
	baseTime := time.Now()

	source.On("Supports", "heart_rate").Return(true)
	source.On("Fetch", context.Background(), "user-1", []string{"heart_rate"}).Return([]domain.BiometricSample{
		{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm", Timestamp: baseTime, Quality: "high", Confidence: 0.9},
	}, nil)
	source.On("DeviceType").Return("test")

	composite := adapters.NewCompositeBiometricSource(source)
	uc := NewFetchBiometricDataUseCase(composite)

	req := FetchBiometricDataRequest{
		UserID:      "user-1",
		MetricTypes: []string{"heart_rate"},
		Sources:     []domain.BiometricSource{source},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Samples, 1)
	assert.Equal(t, 72.0, resp.Samples[0].Value)
	assert.Equal(t, 1, resp.Sources)
	assert.Equal(t, 1, resp.Available)
}

func TestFetchBiometricDataUseCase_Execute_FetchError(t *testing.T) {
	source := newMockBiometricSource()

	source.On("Supports", "heart_rate").Return(true)
	source.On("Fetch", context.Background(), "user-1", []string{"heart_rate"}).Return([]domain.BiometricSample(nil), errors.New("fetch failed"))
	source.On("DeviceType").Return("test")

	composite := adapters.NewCompositeBiometricSource(source)
	uc := NewFetchBiometricDataUseCase(composite)

	req := FetchBiometricDataRequest{
		UserID:      "user-1",
		MetricTypes: []string{"heart_rate"},
		Sources:     []domain.BiometricSource{source},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "fetch biometric data")
}

func TestNewValidateTrainingPlanUseCase(t *testing.T) {
	mockRepo := &mockMedicalService{}
	ms := medical.NewMedicalService(mockRepo, nil)
	uc := NewValidateTrainingPlanUseCase(ms)
	assert.NotNil(t, uc)
	assert.NotNil(t, uc.medicalService)
}

func TestValidateTrainingPlanUseCase_Execute_NilWorkout(t *testing.T) {
	mockRepo := &mockMedicalService{}
	ms := medical.NewMedicalService(mockRepo, nil)
	uc := NewValidateTrainingPlanUseCase(ms)

	req := ValidateTrainingPlanRequest{
		UserID:  "user-1",
		Workout: nil,
	}

	resp, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "workout plan is required", err.Error())
}

func TestValidateTrainingPlanUseCase_Execute_Success(t *testing.T) {
	mockRepo := &mockMedicalService{}
	ms := medical.NewMedicalService(mockRepo, nil)
	uc := NewValidateTrainingPlanUseCase(ms)

	workout := &domain.WorkoutPlan{
		UserID:          "user-1",
		Name:            "Morning Cardio",
		Type:            "cardio",
		DurationMinutes: 30,
		MaxHeartRate:    160,
	}

	mockRepo.On("GetActiveConstraints", context.Background(), "user-1").Return([]domain.MedicalConstraint{}, nil)

	req := ValidateTrainingPlanRequest{
		UserID:  "user-1",
		Workout: workout,
	}

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Empty(t, resp.Violations)
	assert.False(t, resp.RequiresReview)
	assert.True(t, resp.Allowed)
}

func TestValidateTrainingPlanUseCase_Execute_WithViolations(t *testing.T) {
	mockRepo := &mockMedicalService{}
	ms := medical.NewMedicalService(mockRepo, nil)
	uc := NewValidateTrainingPlanUseCase(ms)

	workout := &domain.WorkoutPlan{
		UserID:          "user-1",
		Name:            "Intense Cardio",
		Type:            "cardio",
		DurationMinutes: 60,
		MaxHeartRate:    180,
	}

	constraints := []domain.MedicalConstraint{
		{
			ID:    "mc-1",
			Code:  "I10",
			Label: "Hypertension",
			ImpactOnTraining: []domain.ImpactRule{
				{
					Metric:       "heart_rate_max",
					ThresholdMax: func() *float64 { v := 160.0; return &v }(),
					Action:       "caution",
				},
			},
		},
	}

	mockRepo.On("GetActiveConstraints", context.Background(), "user-1").Return(constraints, nil)

	req := ValidateTrainingPlanRequest{
		UserID:  "user-1",
		Workout: workout,
	}

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Violations, 1)
	assert.False(t, resp.RequiresReview)
	assert.True(t, resp.Allowed)
}

func TestValidateTrainingPlanUseCase_Execute_AvoidAction(t *testing.T) {
	mockRepo := &mockMedicalService{}
	ms := medical.NewMedicalService(mockRepo, nil)
	uc := NewValidateTrainingPlanUseCase(ms)

	workout := &domain.WorkoutPlan{
		UserID:          "user-1",
		Name:            "Intense Cardio",
		Type:            "cardio",
		DurationMinutes: 60,
		MaxHeartRate:    180,
	}

	constraints := []domain.MedicalConstraint{
		{
			ID:    "mc-1",
			Code:  "I21",
			Label: "Acute myocardial infarction",
			ImpactOnTraining: []domain.ImpactRule{
				{
					Metric:       "heart_rate_max",
					ThresholdMax: func() *float64 { v := 120.0; return &v }(),
					Action:       "avoid",
				},
			},
		},
	}

	mockRepo.On("GetActiveConstraints", context.Background(), "user-1").Return(constraints, nil)

	req := ValidateTrainingPlanRequest{
		UserID:  "user-1",
		Workout: workout,
	}

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Violations, 1)
	assert.False(t, resp.RequiresReview)
	assert.False(t, resp.Allowed)
}

func TestNewImportDeviceDataUseCase(t *testing.T) {
	source := newMockBiometricSource()
	var sink domain.BiometricSink
	uc := NewImportDeviceDataUseCase(source, sink)
	assert.NotNil(t, uc)
	assert.Equal(t, source, uc.source)
	assert.Nil(t, uc.sink)
}

func TestImportDeviceDataUseCase_Execute_EmptyDeviceID(t *testing.T) {
	source := newMockBiometricSource()
	var sink domain.BiometricSink
	uc := NewImportDeviceDataUseCase(source, sink)

	req := ImportDeviceDataRequest{
		UserID:      "user-1",
		DeviceID:    "",
		MetricTypes: []string{"heart_rate"},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, "device ID is required", err.Error())
}

func TestImportDeviceDataUseCase_Execute_Success(t *testing.T) {
	source := newMockBiometricSource()
	sink := &mockBiometricSink{}
	baseTime := time.Now()

	source.On("Supports", "heart_rate").Return(true)
	source.On("Fetch", context.Background(), "user-1", []string{"heart_rate"}).Return([]domain.BiometricSample{
		{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm", Timestamp: baseTime, Quality: "high", Confidence: 0.9},
	}, nil)
	sink.On("Store", context.Background(), mock.Anything).Return(nil)

	uc := NewImportDeviceDataUseCase(source, sink)

	req := ImportDeviceDataRequest{
		UserID:      "user-1",
		DeviceID:    "device-1",
		MetricTypes: []string{"heart_rate"},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, resp.Imported)
	assert.Empty(t, resp.Errors)
}

func TestImportDeviceDataUseCase_Execute_NoData(t *testing.T) {
	source := newMockBiometricSource()
	var sink domain.BiometricSink

	source.On("Supports", "heart_rate").Return(true)
	source.On("Fetch", context.Background(), "user-1", []string{"heart_rate"}).Return([]domain.BiometricSample{}, nil)

	uc := NewImportDeviceDataUseCase(source, sink)

	req := ImportDeviceDataRequest{
		UserID:      "user-1",
		DeviceID:    "device-1",
		MetricTypes: []string{"heart_rate"},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 0, resp.Imported)
	assert.Equal(t, []string{"no data available from device"}, resp.Errors)
}

func TestImportDeviceDataUseCase_Execute_FetchError(t *testing.T) {
	source := newMockBiometricSource()
	var sink domain.BiometricSink

	source.On("Supports", "heart_rate").Return(true)
	source.On("Fetch", context.Background(), "user-1", []string{"heart_rate"}).Return([]domain.BiometricSample(nil), errors.New("device unavailable"))

	uc := NewImportDeviceDataUseCase(source, sink)

	req := ImportDeviceDataRequest{
		UserID:      "user-1",
		DeviceID:    "device-1",
		MetricTypes: []string{"heart_rate"},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "fetch device data")
}

func TestImportDeviceDataUseCase_Execute_StoreError(t *testing.T) {
	source := newMockBiometricSource()
	sink := &mockBiometricSink{}
	baseTime := time.Now()

	source.On("Supports", "heart_rate").Return(true)
	source.On("Fetch", context.Background(), "user-1", []string{"heart_rate"}).Return([]domain.BiometricSample{
		{UserID: "user-1", MetricType: "heart_rate", Value: 72, Unit: "bpm", Timestamp: baseTime, Quality: "high", Confidence: 0.9},
	}, nil)
	sink.On("Store", context.Background(), mock.Anything).Return(errors.New("database error"))

	uc := NewImportDeviceDataUseCase(source, sink)

	req := ImportDeviceDataRequest{
		UserID:      "user-1",
		DeviceID:    "device-1",
		MetricTypes: []string{"heart_rate"},
	}

	resp, err := uc.Execute(context.Background(), req)
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "store device data")
}
