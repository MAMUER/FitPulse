// Package port defines repository and service interfaces.
package port

import (
	"context"
	"time"

	"github.com/MAMUER/project/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int) ([]*entity.User, error)
	Count(ctx context.Context) (int, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ListByRole(ctx context.Context, role string, page, pageSize int) ([]*entity.User, int, error)
	BanUser(ctx context.Context, id string) error
	UnbanUser(ctx context.Context, id string) error
}

type BiometricRepository interface {
	Create(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	BatchCreate(ctx context.Context, records []*entity.BiometricRecord) (int, error)
	GetByUserID(ctx context.Context, userID, metricType string, limit, offset int) ([]*entity.BiometricRecord, error)
	GetLatest(ctx context.Context, userID, metricType string) (*entity.BiometricRecord, error)
	Update(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	Delete(ctx context.Context, id string) error
}

type TrainingRepository interface {
	CreatePlan(ctx context.Context, plan *entity.TrainingPlan) (*entity.TrainingPlan, error)
	GetPlan(ctx context.Context, userID, planID string) (*entity.TrainingPlan, error)
	ListPlans(ctx context.Context, userID string, page, pageSize int) ([]*entity.TrainingPlan, int, error)
	CompleteWorkout(ctx context.Context, userID, planID string) error
	GetProgress(ctx context.Context, userID string) (map[string]interface{}, error)
	GetAchievements(ctx context.Context, userID string) ([]*entity.Achievement, error)
}

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID string) (*entity.User, error)
	UpdateProfile(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error
	UserExists(ctx context.Context, userID string) (bool, error)
	CreateProfile(ctx context.Context, userID string) error
	UpsertProfile(ctx context.Context, userID string, data *ProfileData) error
}

type ProfileData struct {
	Age          int32
	Gender       string
	HeightCm     int32
	WeightKg     float64
	FitnessLevel string
	Nutrition    string
	SleepHours   float32
}

type InviteRepository interface {
	Create(ctx context.Context, invite *Invite) error
	GetByCode(ctx context.Context, code string) (*Invite, error)
	List(ctx context.Context, page, pageSize int) ([]*Invite, int, error)
	Revoke(ctx context.Context, code string) error
}

type InviteCodeRepository interface {
	List(ctx context.Context, page, pageSize int) ([]*InviteCode, int, error)
	Create(ctx context.Context, invite *InviteCode) error
	Revoke(ctx context.Context, code string) error
	Validate(ctx context.Context, code string) (*InviteCode, error)
	UseInviteCode(ctx context.Context, code string) error
	ValidateInviteCodeUse(ctx context.Context, code string) (bool, string, string, string, error)
	LogInviteCodeUse(ctx context.Context, code, userID string) error
}

type InviteCode struct {
	Code      string
	Role      string
	Specialty *string
	MaxUses   int
	UsedCount int
	IsActive  bool
	CreatedBy string
	CreatedAt time.Time
}

type Invite struct {
	Code      string
	Role      string
	Specialty string
	MaxUses   int
	UsedCount int
	IsActive  bool
	CreatedAt time.Time
	InviteURL string
}

type HealthConditionRepository interface {
	Create(ctx context.Context, condition *entity.HealthCondition) (*entity.HealthCondition, error)
	List(ctx context.Context, userID, conditionType string) ([]*entity.HealthCondition, error)
	Delete(ctx context.Context, id string) error
}

type UserHealthConditionRepository interface {
	List(ctx context.Context, userID string) ([]*UserHealthCondition, error)
	Upsert(ctx context.Context, condition *UserHealthCondition) (*UserHealthCondition, error)
	Delete(ctx context.Context, id, userID string) error
}

type UserHealthCondition struct {
	ID            string
	UserID        string
	ConditionType string
	ConditionName string
	Severity      string
	DiagnosedAt   *time.Time
	IsActive      bool
	Notes         *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BodyCompositionRepository interface {
	Create(ctx context.Context, bc *entity.BodyComposition) (*entity.BodyComposition, error)
	List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*entity.BodyComposition, error)
}

type UserBodyCompositionRepository interface {
	List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*UserBodyComposition, error)
	Create(ctx context.Context, bc *UserBodyComposition) (*UserBodyComposition, error)
}

type UserBodyComposition struct {
	ID                   string
	UserID               string
	RecordedAt           time.Time
	WeightKG             float64
	HeightCM             float64
	BMI                  float64
	BodyFatPercentage    *float64
	MuscleMassPercentage *float64
	BoneMassPercentage   *float64
	WaterPercentage      *float64
	VisceralFatRating    *float64
	MetabolicAge         *float64
	Source               string
	CreatedAt            time.Time
}

type MenstrualCycleRepository interface {
	Create(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	List(ctx context.Context, userID string) ([]*entity.MenstrualCycle, error)
	Update(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	Delete(ctx context.Context, id string) error
}

type UserMenstrualRepository interface {
	ListCycles(ctx context.Context, userID string) ([]*UserMenstrualCycle, error)
	CreateCycle(ctx context.Context, cycle *UserMenstrualCycle) (*UserMenstrualCycle, error)
	CreateCycleWithDetails(ctx context.Context, cycle *UserMenstrualCycle) (*UserMenstrualCycle, error)
	UpdateCycle(ctx context.Context, cycle *UserMenstrualCycle) (*UserMenstrualCycle, error)
	UpdateCycleWithDetails(ctx context.Context, cycle *UserMenstrualCycle) (*UserMenstrualCycle, error)
	DeleteCycle(ctx context.Context, id, userID string) error
	ListSymptoms(ctx context.Context, cycleID string) ([]string, error)
	CreateSymptom(ctx context.Context, cycleID, symptom string) error
	DeleteSymptoms(ctx context.Context, cycleID string) error
	ListMoods(ctx context.Context, cycleID string) ([]string, error)
	CreateMood(ctx context.Context, cycleID, mood string) error
	DeleteMoods(ctx context.Context, cycleID string) error
}

type UserMenstrualCycle struct {
	ID             string
	UserID         string
	CycleStartDate string
	CycleEndDate   string
	FlowIntensity  string
	Notes          string
	Symptoms       []string
	Moods          []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AchievementRepository interface {
	Create(ctx context.Context, achievement *entity.Achievement) (*entity.Achievement, error)
	List(ctx context.Context, userID string) ([]*entity.Achievement, error)
}

type AchievementRepositoryEx interface {
	ListWithEarnedStatus(ctx context.Context, userID string) ([]*AchievementInfo, error)
	Earn(ctx context.Context, userID, achievementID string) error
}

type AchievementInfo struct {
	ID          string
	Name        string
	Description string
	IconURL     string
	EarnedAt    *time.Time
	CreatedAt   time.Time
}

type DeviceRepository interface {
	List(ctx context.Context, userID string) ([]*entity.Device, error)
	Create(ctx context.Context, device *entity.Device) (*entity.Device, error)
	Delete(ctx context.Context, userID, deviceID string) error
}

type EmailVerification struct {
	ID        string
	UserID    string
	Email     string
	EmailHash string
	Token     string
	Used      bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

type EmailVerificationRepository interface {
	Create(ctx context.Context, ev *EmailVerification) error
	GetValidToken(ctx context.Context, token string) (*EmailVerification, error)
	GetByUserID(ctx context.Context, userID string) (*EmailVerification, error)
	MarkUsed(ctx context.Context, token string) error
	MarkUserEmailVerified(ctx context.Context, userID string) error
}

type RefreshToken struct {
	ID        string
	UserID    string
	Token     string
	Revoked   bool
	ExpiresAt time.Time
	CreatedAt time.Time
}

type RefreshTokenRepository interface {
	GetValid(ctx context.Context, token string) (*RefreshToken, error)
	Create(ctx context.Context, rt *RefreshToken) error
	MarkRevoked(ctx context.Context, token string) error
}

type ExternalSecret struct {
	ID        string
	Name      string
	Value     string
	CreatedAt string
	UpdatedAt string
}

type ExternalSecretRepository interface {
	GetByName(ctx context.Context, name string) (*ExternalSecret, error)
	List(ctx context.Context) ([]*ExternalSecret, error)
	Upsert(ctx context.Context, secret *ExternalSecret) error
	Delete(ctx context.Context, name string) error
}
