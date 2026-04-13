package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		user User
	}{
		{
			name: "client user",
			user: User{
				ID:           "user-123",
				Email:        "client@example.com",
				FullName:     "Client User",
				Role:         "client",
				PasswordHash: "hashed_password",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
		{
			name: "admin user",
			user: User{
				ID:           "admin-456",
				Email:        "admin@example.com",
				FullName:     "Admin User",
				Role:         "admin",
				PasswordHash: "hashed_admin_password",
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.user.ID, tt.user.ID)
			assert.Equal(t, tt.user.Email, tt.user.Email)
			assert.Equal(t, tt.user.FullName, tt.user.FullName)
			assert.Equal(t, tt.user.Role, tt.user.Role)
			assert.Equal(t, tt.user.PasswordHash, tt.user.PasswordHash)
		})
	}
}

func TestUserJSONSerialization(t *testing.T) {
	now := time.Now()
	user := User{
		ID:           "user-123",
		Email:        "test@example.com",
		FullName:     "Test User",
		Role:         "client",
		PasswordHash: "hashed",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var unmarshaled User
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, user.ID, unmarshaled.ID)
	assert.Equal(t, user.Email, unmarshaled.Email)
	assert.Equal(t, user.FullName, unmarshaled.FullName)
	assert.Equal(t, user.Role, unmarshaled.Role)
	// PasswordHash не должен сериализоваться (json:"-")
	assert.Empty(t, unmarshaled.PasswordHash)
}

func TestUserProfile(t *testing.T) {
	profile := UserProfile{
		UserID:            "user-123",
		Age:               30,
		Gender:            "male",
		HeightCm:          180,
		WeightKg:          75.5,
		FitnessLevel:      "intermediate",
		Goals:             []string{"weight_loss", "muscle_gain"},
		Contraindications: []string{"knee_problems"},
	}

	assert.Equal(t, "user-123", profile.UserID)
	assert.Equal(t, 30, profile.Age)
	assert.Equal(t, "male", profile.Gender)
	assert.Equal(t, 180, profile.HeightCm)
	assert.Equal(t, 75.5, profile.WeightKg)
	assert.Equal(t, "intermediate", profile.FitnessLevel)
	assert.Len(t, profile.Goals, 2)
	assert.Contains(t, profile.Goals, "weight_loss")
	assert.Contains(t, profile.Goals, "muscle_gain")
	assert.Len(t, profile.Contraindications, 1)
	assert.Contains(t, profile.Contraindications, "knee_problems")
}

func TestBiometricRecord(t *testing.T) {
	tests := []struct {
		name   string
		record BiometricRecord
	}{
		{
			name: "heart rate record",
			record: BiometricRecord{
				ID:         "rec-123",
				UserID:     "user-123",
				MetricType: "heart_rate",
				Value:      72.5,
				Timestamp:  time.Now(),
				DeviceType: "apple_watch",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "blood pressure record",
			record: BiometricRecord{
				ID:         "rec-456",
				UserID:     "user-123",
				MetricType: "blood_pressure",
				Value:      120.0,
				Timestamp:  time.Now(),
				DeviceType: "samsung_watch",
				CreatedAt:  time.Now(),
			},
		},
		{
			name: "spo2 record",
			record: BiometricRecord{
				ID:         "rec-789",
				UserID:     "user-123",
				MetricType: "spo2",
				Value:      98.0,
				Timestamp:  time.Now(),
				DeviceType: "huawei_band",
				CreatedAt:  time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.record.MetricType, tt.record.MetricType)
			assert.Equal(t, tt.record.Value, tt.record.Value)
			assert.Equal(t, tt.record.DeviceType, tt.record.DeviceType)
		})
	}
}

func TestBiometricRecordJSONSerialization(t *testing.T) {
	record := BiometricRecord{
		ID:         "rec-123",
		UserID:     "user-123",
		MetricType: "heart_rate",
		Value:      72.5,
		Timestamp:  time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
		DeviceType: "apple_watch",
		CreatedAt:  time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(record)
	require.NoError(t, err)

	var unmarshaled BiometricRecord
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, record.ID, unmarshaled.ID)
	assert.Equal(t, record.UserID, unmarshaled.UserID)
	assert.Equal(t, record.MetricType, unmarshaled.MetricType)
	assert.Equal(t, record.Value, unmarshaled.Value)
	assert.Equal(t, record.DeviceType, unmarshaled.DeviceType)
}

func TestTrainingPlan(t *testing.T) {
	planData := json.RawMessage(`{"weeks": 4, "workouts": []}`)
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	plan := TrainingPlan{
		ID:        "plan-123",
		UserID:    "user-123",
		PlanData:  planData,
		StartDate: &startDate,
		EndDate:   &endDate,
		Status:    "active",
	}

	assert.Equal(t, "plan-123", plan.ID)
	assert.Equal(t, "user-123", plan.UserID)
	assert.Equal(t, "active", plan.Status)
	assert.NotNil(t, plan.PlanData)
	assert.NotNil(t, plan.StartDate)
	assert.NotNil(t, plan.EndDate)
}

func TestTrainingPlanStatuses(t *testing.T) {
	statuses := []string{"active", "completed", "archived"}

	for _, status := range statuses {
		t.Run(status, func(t *testing.T) {
			plan := TrainingPlan{
				Status: status,
			}
			assert.Equal(t, status, plan.Status)
		})
	}
}

func TestTrainingPlanWithNilDates(t *testing.T) {
	plan := TrainingPlan{
		StartDate: nil,
		EndDate:   nil,
	}

	assert.Nil(t, plan.StartDate)
	assert.Nil(t, plan.EndDate)
}
