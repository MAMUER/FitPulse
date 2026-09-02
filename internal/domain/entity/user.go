package entity

import "time"

type User struct {
	ID            string
	Email         string
	PasswordHash  string
	FullName      string
	Role          string
	EmailVerified bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
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
