// Package service defines application service interfaces.
package service

import (
	"context"
	"time"

	"github.com/MAMUER/project/internal/domain/entity"
)

type UserService interface {
	Register(ctx context.Context, email, password, fullName, role string) (*entity.User, error)
	Login(ctx context.Context, email, password string) (*entity.User, string, error)
	GetProfile(ctx context.Context, userID string) (*entity.User, error)
	UpdateProfile(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error
	DeleteProfile(ctx context.Context, userID, password string) error
	ConfirmEmail(ctx context.Context, token string) error
	CreateInvite(ctx context.Context, role, specialty string, maxUses int) (string, error)
	ValidateInvite(ctx context.Context, code string) (string, string, error)
	ListUsers(ctx context.Context, page, pageSize int) ([]*entity.User, int, error)
	ListDevices(ctx context.Context, userID string) ([]*entity.Device, error)
	AddDevice(ctx context.Context, device *entity.Device) (*entity.Device, error)
	RemoveDevice(ctx context.Context, userID, deviceID string) error
	SaveSurvey(ctx context.Context, userID string, survey map[string]interface{}, completed bool, completedAt *time.Time) error
	LoadSurvey(ctx context.Context, userID string) (map[string]interface{}, bool, *time.Time, error)
}

type BiometricService interface {
	AddRecord(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	BatchAddRecords(ctx context.Context, records []*entity.BiometricRecord) (int, error)
	GetRecords(ctx context.Context, userID, metricType string, limit int) ([]*entity.BiometricRecord, error)
	GetLatest(ctx context.Context, userID, metricType string) (*entity.BiometricRecord, error)
	UpdateRecord(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	DeleteRecord(ctx context.Context, id string) error
}

type TrainingService interface {
	GeneratePlan(ctx context.Context, userID, classification string, durationWeeks int, availableDays []int) (*entity.TrainingPlan, error)
	GetPlan(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error)
	ListPlans(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error)
	CompleteWorkout(ctx context.Context, userID, planID string, rating int, feedback string) error
	GetProgress(ctx context.Context, userID string) (map[string]interface{}, error)
	GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error)
}
