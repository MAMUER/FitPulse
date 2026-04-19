package validator

import (
	"testing"

	biometricpb "github.com/MAMUER/Project/api/gen/biometric"
	trainingpb "github.com/MAMUER/Project/api/gen/training"
	userpb "github.com/MAMUER/Project/api/gen/user"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ========== Biometric Validator Tests ==========

func TestValidateBiometricRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *biometricpb.AddRecordRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errMsg:  "request is nil",
		},
		{
			name: "valid heart rate",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantErr: false,
		},
		{
			name: "missing user_id",
			req: &biometricpb.AddRecordRequest{
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantErr: true,
			errMsg:  "user_id is required",
		},
		{
			name: "missing metric_type",
			req: &biometricpb.AddRecordRequest{
				UserId: "user-123",
				Value:  75.0,
			},
			wantErr: true,
			errMsg:  "metric_type is required",
		},
		{
			name: "negative value",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      -10.0,
			},
			wantErr: true,
			errMsg:  "value cannot be negative",
		},
		{
			name: "heart_rate too low",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      25.0,
			},
			wantErr: true,
			errMsg:  "heart_rate out of valid range",
		},
		{
			name: "heart_rate too high",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      250.0,
			},
			wantErr: true,
			errMsg:  "heart_rate out of valid range",
		},
		{
			name: "heart_rate boundary low",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      30.0,
			},
			wantErr: false,
		},
		{
			name: "heart_rate boundary high",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "heart_rate",
				Value:      220.0,
			},
			wantErr: false,
		},
		{
			name: "spo2 valid",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      98.0,
			},
			wantErr: false,
		},
		{
			name: "spo2 too low",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      69.0,
			},
			wantErr: true,
			errMsg:  "spo2 out of valid range",
		},
		{
			name: "spo2 boundary low",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      70.0,
			},
			wantErr: false,
		},
		{
			name: "spo2 boundary high",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "spo2",
				Value:      100.0,
			},
			wantErr: false,
		},
		{
			name: "unknown metric - passes validation",
			req: &biometricpb.AddRecordRequest{
				UserId:     "user-123",
				MetricType: "unknown_metric",
				Value:      50.0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBiometricRequest(tt.req)
			if !tt.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidateBiometricRecord(t *testing.T) {
	tests := []struct {
		name    string
		req     *biometricpb.AddRecordRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: true,
			errMsg:  "request is nil",
		},
		{
			name: "valid without user_id",
			req: &biometricpb.AddRecordRequest{
				MetricType: "heart_rate",
				Value:      75.0,
			},
			wantErr: false,
		},
		{
			name: "missing metric_type",
			req: &biometricpb.AddRecordRequest{
				Value: 75.0,
			},
			wantErr: true,
			errMsg:  "metric_type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBiometricRecord(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ========== User Validator Tests ==========

func TestValidateRegisterRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *userpb.RegisterRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid registration",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "securepass123",
				FullName: "Test User",
				Role:     "client",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing email",
			req: &userpb.RegisterRequest{
				Email:    "",
				Password: "securepass123",
				FullName: "Test User",
				Role:     "client",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "email is required",
		},
		{
			name: "invalid email format",
			req: &userpb.RegisterRequest{
				Email:    "not-an-email",
				Password: "securepass123",
				FullName: "Test User",
				Role:     "client",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "invalid email format",
		},
		{
			name: "missing password",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "",
				FullName: "Test User",
				Role:     "client",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "password is required",
		},
		{
			name: "password too short",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "short",
				FullName: "Test User",
				Role:     "client",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "password must be at least 8 characters",
		},
		{
			name: "password exactly 8 chars",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "12345678",
				FullName: "Test User",
				Role:     "client",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing full name",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "securepass123",
				FullName: "",
				Role:     "client",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "full name is required",
		},
		{
			name: "missing role",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "securepass123",
				FullName: "Test User",
				Role:     "",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "role is required",
		},
		{
			name: "invalid role",
			req: &userpb.RegisterRequest{
				Email:    "test@example.com",
				Password: "securepass123",
				FullName: "Test User",
				Role:     "hacker",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "invalid role",
		},
		{
			name: "valid admin role",
			req: &userpb.RegisterRequest{
				Email:    "admin@example.com",
				Password: "securepass123",
				FullName: "Admin User",
				Role:     "admin",
			},
			wantCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegisterRequest(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

func TestValidateLoginRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *userpb.LoginRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid login",
			req: &userpb.LoginRequest{
				Email:    "test@example.com",
				Password: "securepass123",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing email",
			req: &userpb.LoginRequest{
				Email:    "",
				Password: "securepass123",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "email is required",
		},
		{
			name: "missing password",
			req: &userpb.LoginRequest{
				Email:    "test@example.com",
				Password: "",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoginRequest(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

func TestValidateProfileUpdate(t *testing.T) {
	age := int32(30)
	gender := "male"
	height := int32(180)
	weight := 75.0
	fitness := "intermediate"

	tests := []struct {
		name     string
		req      *userpb.UpdateProfileRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid update",
			req: &userpb.UpdateProfileRequest{
				UserId:       "user-123",
				Age:          &age,
				Gender:       &gender,
				HeightCm:     &height,
				WeightKg:     &weight,
				FitnessLevel: &fitness,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing user_id",
			req: &userpb.UpdateProfileRequest{
				UserId: "",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "user_id is required",
		},
		{
			name: "age negative",
			req: &userpb.UpdateProfileRequest{
				UserId: "user-123",
				Age:    ptrInt32(-1),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "age must be between 0 and 150",
		},
		{
			name: "age too high",
			req: &userpb.UpdateProfileRequest{
				UserId: "user-123",
				Age:    ptrInt32(200),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "age must be between 0 and 150",
		},
		{
			name: "height too low",
			req: &userpb.UpdateProfileRequest{
				UserId:   "user-123",
				HeightCm: ptrInt32(30),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "height_cm must be between 50 and 300",
		},
		{
			name: "height too high",
			req: &userpb.UpdateProfileRequest{
				UserId:   "user-123",
				HeightCm: ptrInt32(350),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "height_cm must be between 50 and 300",
		},
		{
			name: "weight too low",
			req: &userpb.UpdateProfileRequest{
				UserId:   "user-123",
				WeightKg: ptrFloat64(0.5),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "weight_kg must be between 1 and 500",
		},
		{
			name: "weight too high",
			req: &userpb.UpdateProfileRequest{
				UserId:   "user-123",
				WeightKg: ptrFloat64(600),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "weight_kg must be between 1 and 500",
		},
		{
			name: "invalid fitness level",
			req: &userpb.UpdateProfileRequest{
				UserId:       "user-123",
				FitnessLevel: ptrString("extreme"),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "fitness_level must be beginner, intermediate, or advanced",
		},
		{
			name: "invalid gender",
			req: &userpb.UpdateProfileRequest{
				UserId: "user-123",
				Gender: ptrString("attack"),
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "gender must be male, female, or other",
		},
		{
			name: "only user_id - all optional fields nil",
			req: &userpb.UpdateProfileRequest{
				UserId: "user-123",
			},
			wantCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProfileUpdate(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

// ========== Training Validator Tests ==========

func TestValidateGeneratePlanRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *trainingpb.GeneratePlanRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid plan request",
			req: &trainingpb.GeneratePlanRequest{
				UserId:              "user-123",
				DurationWeeks:       4,
				AvailableDays:       []int32{1, 3, 5},
				ClassificationClass: "endurance",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing user_id",
			req: &trainingpb.GeneratePlanRequest{
				DurationWeeks: 4,
				AvailableDays: []int32{1, 3, 5},
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "user_id is required",
		},
		{
			name: "duration weeks zero - uses default",
			req: &trainingpb.GeneratePlanRequest{
				UserId:        "user-123",
				DurationWeeks: 0,
				AvailableDays: []int32{1, 3, 5},
			},
			wantCode: codes.OK,
		},
		{
			name: "duration weeks negative",
			req: &trainingpb.GeneratePlanRequest{
				UserId:        "user-123",
				DurationWeeks: -1,
				AvailableDays: []int32{1, 3, 5},
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "duration_weeks must be greater than 0",
		},
		{
			name: "duration weeks too large",
			req: &trainingpb.GeneratePlanRequest{
				UserId:        "user-123",
				DurationWeeks: 100,
				AvailableDays: []int32{1, 3, 5},
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "duration_weeks must not exceed 52",
		},
		{
			name: "duration weeks at max boundary",
			req: &trainingpb.GeneratePlanRequest{
				UserId:        "user-123",
				DurationWeeks: 52,
				AvailableDays: []int32{1, 3, 5},
			},
			wantCode: codes.OK,
		},
		{
			name: "missing available days",
			req: &trainingpb.GeneratePlanRequest{
				UserId:        "user-123",
				DurationWeeks: 4,
				AvailableDays: []int32{},
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "available_days is required",
		},
		{
			name: "too many available days",
			req: &trainingpb.GeneratePlanRequest{
				UserId:        "user-123",
				DurationWeeks: 4,
				AvailableDays: []int32{1, 2, 3, 4, 5, 6, 7, 8},
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "available_days must not exceed 7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGeneratePlanRequest(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

func TestValidateCompleteWorkoutRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *trainingpb.CompleteWorkoutRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid request",
			req: &trainingpb.CompleteWorkoutRequest{
				UserId:    "user-123",
				PlanId:    "plan-456",
				WorkoutId: "workout-789",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing user_id",
			req: &trainingpb.CompleteWorkoutRequest{
				PlanId:    "plan-456",
				WorkoutId: "workout-789",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "user_id is required",
		},
		{
			name: "missing plan_id",
			req: &trainingpb.CompleteWorkoutRequest{
				UserId:    "user-123",
				WorkoutId: "workout-789",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "plan_id is required",
		},
		{
			name: "missing workout_id",
			req: &trainingpb.CompleteWorkoutRequest{
				UserId: "user-123",
				PlanId: "plan-456",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "workout_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCompleteWorkoutRequest(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

func TestValidateListPlansRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *trainingpb.ListPlansRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid request",
			req: &trainingpb.ListPlansRequest{
				UserId:   "user-123",
				Page:     0,
				PageSize: 10,
			},
			wantCode: codes.OK,
		},
		{
			name: "missing user_id",
			req: &trainingpb.ListPlansRequest{
				Page:     0,
				PageSize: 10,
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "user_id is required",
		},
		{
			name: "page size zero",
			req: &trainingpb.ListPlansRequest{
				UserId:   "user-123",
				Page:     0,
				PageSize: 0,
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "page_size must be greater than 0",
		},
		{
			name: "negative page",
			req: &trainingpb.ListPlansRequest{
				UserId:   "user-123",
				Page:     -1,
				PageSize: 10,
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "page must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateListPlansRequest(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

func TestValidateGetProgressRequest(t *testing.T) {
	tests := []struct {
		name     string
		req      *trainingpb.GetProgressRequest
		wantCode codes.Code
		errMsg   string
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			errMsg:   "request is nil",
		},
		{
			name: "valid request",
			req: &trainingpb.GetProgressRequest{
				UserId: "user-123",
			},
			wantCode: codes.OK,
		},
		{
			name: "missing user_id",
			req: &trainingpb.GetProgressRequest{
				UserId: "",
			},
			wantCode: codes.InvalidArgument,
			errMsg:   "user_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGetProgressRequest(tt.req)
			if tt.wantCode == codes.OK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				st, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, tt.wantCode, st.Code())
				assert.Contains(t, st.Message(), tt.errMsg)
			}
		})
	}
}

// Helper functions
func ptrInt32(v int32) *int32       { return &v }
func ptrString(v string) *string    { return &v }
func ptrFloat64(v float64) *float64 { return &v }
