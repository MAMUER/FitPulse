package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSubscriptionPlanPriceMonthly(t *testing.T) {
	price, ok := SubscriptionPlanPrice("monthly")
	assert.True(t, ok)
	assert.Equal(t, 999.0, price)
}

func TestSubscriptionPlanPriceQuarterly(t *testing.T) {
	price, ok := SubscriptionPlanPrice("quarterly")
	assert.True(t, ok)
	assert.Equal(t, 2499.0, price)
}

func TestSubscriptionPlanPriceYearly(t *testing.T) {
	price, ok := SubscriptionPlanPrice("yearly")
	assert.True(t, ok)
	assert.Equal(t, 7999.0, price)
}

func TestSubscriptionPlanPriceUnknown(t *testing.T) {
	_, ok := SubscriptionPlanPrice("unknown")
	assert.False(t, ok)
}

func TestSubscriptionPlanPriceEmpty(t *testing.T) {
	_, ok := SubscriptionPlanPrice("")
	assert.False(t, ok)
}

func TestSubscriptionPlanDurationMonthly(t *testing.T) {
	d, ok := SubscriptionPlanDuration("monthly")
	assert.True(t, ok)
	assert.Equal(t, 30*24*time.Hour, d)
}

func TestSubscriptionPlanDurationQuarterly(t *testing.T) {
	d, ok := SubscriptionPlanDuration("quarterly")
	assert.True(t, ok)
	assert.Equal(t, 90*24*time.Hour, d)
}

func TestSubscriptionPlanDurationYearly(t *testing.T) {
	d, ok := SubscriptionPlanDuration("yearly")
	assert.True(t, ok)
	assert.Equal(t, 365*24*time.Hour, d)
}

func TestSubscriptionPlanDurationUnknown(t *testing.T) {
	_, ok := SubscriptionPlanDuration("unknown")
	assert.False(t, ok)
}

func TestSubscriptionPlanDurationEmpty(t *testing.T) {
	_, ok := SubscriptionPlanDuration("")
	assert.False(t, ok)
}

func TestDoctorStruct(t *testing.T) {
	d := Doctor{
		ID:            "doc-1",
		Specialty:     "sports_medicine",
		LicenseNumber: "LIC-001",
		Phone:         "+79001234567",
		Bio:           "Experienced sports doctor",
		IsActive:      true,
	}
	_ = d.Specialty
	_ = d.LicenseNumber
	_ = d.Phone
	_ = d.Bio
	assert.Equal(t, "doc-1", d.ID)
	assert.True(t, d.IsActive)
	assert.Nil(t, d.UserID)
}

func TestSubscriptionStruct(t *testing.T) {
	s := Subscription{
		ID:       "sub-1",
		UserID:   "user-1",
		DoctorID: "doc-1",
		PlanType: "monthly",
		IsActive: true,
		Price:    999.0,
	}
	_ = s.ID
	_ = s.UserID
	_ = s.DoctorID
	_ = s.Price
	assert.True(t, s.IsActive)
	assert.Equal(t, "monthly", s.PlanType)
}

func TestMessageStruct(t *testing.T) {
	m := Message{
		ID:          "msg-1",
		UserID:      "user-1",
		DoctorID:    "doc-1",
		Message:     "Hello",
		MessageType: "text",
		IsRead:      false,
	}
	_ = m.ID
	_ = m.UserID
	_ = m.DoctorID
	_ = m.Message
	_ = m.MessageType
	assert.False(t, m.IsRead)
	assert.Nil(t, m.SenderUserID)
	assert.Nil(t, m.SenderDoctorID)
}

func TestPrescriptionStruct(t *testing.T) {
	p := Prescription{
		ID:               "rx-1",
		UserID:           "user-1",
		DoctorID:         "doc-1",
		PrescriptionType: "medication",
		Title:            "Test prescription",
		Description:      "Take twice daily",
		Priority:         "high",
		Status:           "active",
	}
	_ = p.ID
	_ = p.UserID
	_ = p.DoctorID
	_ = p.PrescriptionType
	_ = p.Title
	_ = p.Description
	_ = p.Priority
	assert.Equal(t, "active", p.Status)
	assert.Nil(t, p.ConsultationID)
}

func TestConsultationStruct(t *testing.T) {
	c := Consultation{
		ID:       "con-1",
		UserID:   "user-1",
		DoctorID: "doc-1",
		Status:   "scheduled",
		Notes:    "",
	}
	_ = c.ID
	_ = c.UserID
	_ = c.DoctorID
	_ = c.Notes
	assert.Equal(t, "scheduled", c.Status)
	assert.Nil(t, c.StartedAt)
	assert.Nil(t, c.EndedAt)
}

func TestTrainingModificationStruct(t *testing.T) {
	tm := TrainingModification{
		ID:               "tm-1",
		DoctorID:         "doc-1",
		TrainingPlanID:   "plan-1",
		ModificationType: "exercise_added",
		OldValue:         "",
		NewValue:         "Push-ups 3x15",
		Reason:           "Improve upper body strength",
	}
	_ = tm.ID
	_ = tm.DoctorID
	_ = tm.TrainingPlanID
	_ = tm.ModificationType
	_ = tm.OldValue
	_ = tm.NewValue
	_ = tm.Reason
	assert.Equal(t, "exercise_added", tm.ModificationType)
	assert.Empty(t, tm.OldValue)
}

func TestDoctorStatsStruct(t *testing.T) {
	stats := DoctorStats{
		DoctorID:          "doc-1",
		ConsultationCount: 42,
		AvgRating:         4.8,
	}
	_ = stats.DoctorID
	assert.Equal(t, int64(42), stats.ConsultationCount)
	assert.Equal(t, 4.8, stats.AvgRating)
}
