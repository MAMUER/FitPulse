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
}

type BiometricRepository interface {
	Create(ctx context.Context, record *entity.BiometricRecord) (*entity.BiometricRecord, error)
	BatchCreate(ctx context.Context, records []*entity.BiometricRecord) (int, error)
	GetByUserID(ctx context.Context, userID string, metricType string, limit, offset int) ([]*entity.BiometricRecord, error)
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
}

type InviteRepository interface {
	Create(ctx context.Context, invite *Invite) error
	GetByCode(ctx context.Context, code string) (*Invite, error)
	List(ctx context.Context, page, pageSize int) ([]*Invite, int, error)
	Revoke(ctx context.Context, code string) error
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

type BodyCompositionRepository interface {
	Create(ctx context.Context, bc *entity.BodyComposition) (*entity.BodyComposition, error)
	List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*entity.BodyComposition, error)
}

type MenstrualCycleRepository interface {
	Create(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	List(ctx context.Context, userID string) ([]*entity.MenstrualCycle, error)
	Update(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	Delete(ctx context.Context, id string) error
}

type AchievementRepository interface {
	Create(ctx context.Context, achievement *entity.Achievement) (*entity.Achievement, error)
	List(ctx context.Context, userID string) ([]*entity.Achievement, error)
}
