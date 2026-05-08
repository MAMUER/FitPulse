// Package models provides shared data models and types.
package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserProfile struct {
	UserID            string    `json:"user_id"`
	Age               int       `json:"age"`
	Gender            string    `json:"gender"`
	HeightCm          int       `json:"height_cm"`
	WeightKg          float64   `json:"weight_kg"`
	FitnessLevel      string    `json:"fitness_level"`
	Goals             []string  `json:"goals"`
	Contraindications []string  `json:"contraindications"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type BiometricRecord struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	MetricType string    `json:"metric_type"`
	Value      float64   `json:"value"`
	Timestamp  time.Time `json:"timestamp"`
	DeviceType string    `json:"device_type"`
	CreatedAt  time.Time `json:"created_at"`
}

type TrainingPlan struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	PlanData    json.RawMessage `json:"plan_data"`
	GeneratedAt time.Time       `json:"generated_at"`
	StartDate   *time.Time      `json:"start_date"`
	EndDate     *time.Time      `json:"end_date"`
	Status      string          `json:"status"`
	Metadata    json.RawMessage `json:"metadata"`
}
