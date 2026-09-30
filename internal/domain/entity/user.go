// Package entity defines core domain entities.
package entity

import "time"

type User struct {
	ID              string
	Email           string
	PasswordHash    string
	FullName        string
	Role            string
	EmailVerified   bool
	Nickname        string
	ProfilePhotoURL string
	Age             int32
	Gender          string
	HeightCm        int32
	WeightKg        float64
	FitnessLevel    string
	Goals           []string
	Nutrition       string
	SleepHours      float32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type BiometricRecord struct {
	ID         string
	UserID     string
	MetricType string
	Value      float64
	Timestamp  time.Time
	DeviceType string
	Source     string
	CreatedAt  time.Time
}

type TrainingPlan struct {
	ID             string
	UserID         string
	Classification string
	DurationWeeks  int
	AvailableDays  []int
	PlanData       map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type HealthCondition struct {
	ID            string
	UserID        string
	ConditionType string
	Description   string
	CreatedAt     time.Time
}

type BodyComposition struct {
	ID         string
	UserID     string
	WeightKG   float64
	HeightCM   float64
	BMI        float64
	RecordedAt time.Time
}

type MenstrualCycle struct {
	ID        string
	UserID    string
	StartDate time.Time
	EndDate   time.Time
	FlowLevel string
	Notes     string
	CreatedAt time.Time
}

type Achievement struct {
	ID          string
	UserID      string
	Type        string
	Title       string
	Description string
	EarnedAt    time.Time
}

type PlanWeek struct {
	ID                   string
	TrainingPlanID       string
	WeekNumber           int32
	TotalTrainingDays    int32
	TotalDurationMinutes int32
	Days                 []*PlanDay
}

type PlanDay struct {
	ID                   string
	WeekID               string
	DayOfWeek            int32
	TrainingDate         time.Time
	TrainingType         string
	IsRestDay            bool
	TotalDurationMinutes int32
	Notes                string
	Exercises            []*PlanExercise
}

type PlanExercise struct {
	ID              string
	DayID           string
	ExerciseName    string
	DurationMinutes int32
	Intensity       float64
	Sets            int32
	Reps            int32
	RestSeconds     int32
	Description     string
	SortOrder       int32
}

type Device struct {
	ID          string
	UserID      string
	DeviceType  string
	DeviceName  string
	Token       string
	IsConnected bool
	LastSync    time.Time
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

type RefreshToken struct {
	ID        string
	UserID    string
	Token     string
	Revoked   bool
	ExpiresAt time.Time
	CreatedAt time.Time
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

type MenstrualSymptom struct {
	ID      string
	CycleID string
	Symptom string
}

type MenstrualMood struct {
	ID      string
	CycleID string
	Mood    string
}

type UserMenstrualCycle struct {
	ID             string
	UserID         string
	CycleStartDate string
	CycleEndDate   string
	FlowIntensity  string
	Notes          string
	CreatedAt      time.Time
	UpdatedAt      time.Time
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
