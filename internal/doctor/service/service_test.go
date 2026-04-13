package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MAMUER/Project/internal/doctor/model"
	"github.com/MAMUER/Project/internal/doctor/port"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===== Mock repositories =====

type mockDoctorRepo struct {
	listFn        func(ctx context.Context, specialty string, page, pageSize int32) ([]model.Doctor, int64, error)
	getByIDFn     func(ctx context.Context, id string) (*model.Doctor, error)
	getByUserIDFn func(ctx context.Context, userID string) (*model.Doctor, error)
	createFn      func(ctx context.Context, d *model.Doctor) error
	updateFn      func(ctx context.Context, d *model.Doctor) error
}

func (m *mockDoctorRepo) List(ctx context.Context, specialty string, page, pageSize int32) ([]model.Doctor, int64, error) {
	return m.listFn(ctx, specialty, page, pageSize)
}
func (m *mockDoctorRepo) GetByID(ctx context.Context, id string) (*model.Doctor, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockDoctorRepo) GetByUserID(ctx context.Context, userID string) (*model.Doctor, error) {
	return m.getByUserIDFn(ctx, userID)
}
func (m *mockDoctorRepo) Create(ctx context.Context, d *model.Doctor) error {
	return m.createFn(ctx, d)
}
func (m *mockDoctorRepo) Update(ctx context.Context, d *model.Doctor) error {
	return m.updateFn(ctx, d)
}

type mockSubscriptionRepo struct {
	createFn             func(ctx context.Context, s *model.Subscription) error
	getByUserAndDoctorFn func(ctx context.Context, userID, doctorID string) (*model.Subscription, error)
	isActiveFn           func(ctx context.Context, userID, doctorID string) (bool, error)
	cancelFn             func(ctx context.Context, userID, doctorID string) error
}

func (m *mockSubscriptionRepo) Create(ctx context.Context, s *model.Subscription) error {
	return m.createFn(ctx, s)
}
func (m *mockSubscriptionRepo) GetByUserAndDoctor(ctx context.Context, userID, doctorID string) (*model.Subscription, error) {
	return m.getByUserAndDoctorFn(ctx, userID, doctorID)
}
func (m *mockSubscriptionRepo) IsActive(ctx context.Context, userID, doctorID string) (bool, error) {
	return m.isActiveFn(ctx, userID, doctorID)
}
func (m *mockSubscriptionRepo) Cancel(ctx context.Context, userID, doctorID string) error {
	return m.cancelFn(ctx, userID, doctorID)
}

type mockMessageRepo struct {
	createFn         func(ctx context.Context, m *model.Message) (*model.Message, error)
	getChatHistoryFn func(ctx context.Context, userID, doctorID string, page, pageSize int32) ([]model.Message, error)
	markAsReadFn     func(ctx context.Context, userID, doctorID string) (int64, error)
	getUnreadCountFn func(ctx context.Context, userID string) (map[string]int64, int64, error)
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *model.Message) (*model.Message, error) {
	return m.createFn(ctx, msg)
}
func (m *mockMessageRepo) GetChatHistory(ctx context.Context, userID, doctorID string, page, pageSize int32) ([]model.Message, error) {
	return m.getChatHistoryFn(ctx, userID, doctorID, page, pageSize)
}
func (m *mockMessageRepo) MarkAsRead(ctx context.Context, userID, doctorID string) (int64, error) {
	return m.markAsReadFn(ctx, userID, doctorID)
}
func (m *mockMessageRepo) GetUnreadCount(ctx context.Context, userID string) (map[string]int64, int64, error) {
	return m.getUnreadCountFn(ctx, userID)
}

type mockPrescriptionRepo struct {
	createFn       func(ctx context.Context, p *model.Prescription) (*model.Prescription, error)
	getByUserFn    func(ctx context.Context, userID, statusFilter string) ([]model.Prescription, error)
	updateStatusFn func(ctx context.Context, id, status string) error
}

func (m *mockPrescriptionRepo) Create(ctx context.Context, p *model.Prescription) (*model.Prescription, error) {
	return m.createFn(ctx, p)
}
func (m *mockPrescriptionRepo) GetByUser(ctx context.Context, userID, statusFilter string) ([]model.Prescription, error) {
	return m.getByUserFn(ctx, userID, statusFilter)
}
func (m *mockPrescriptionRepo) UpdateStatus(ctx context.Context, id, status string) error {
	return m.updateStatusFn(ctx, id, status)
}

type mockConsultationRepo struct {
	createFn    func(ctx context.Context, c *model.Consultation) (*model.Consultation, error)
	getByUserFn func(ctx context.Context, userID, statusFilter string) ([]model.Consultation, error)
	completeFn  func(ctx context.Context, id, notes string) error
}

func (m *mockConsultationRepo) Create(ctx context.Context, c *model.Consultation) (*model.Consultation, error) {
	return m.createFn(ctx, c)
}
func (m *mockConsultationRepo) GetByUser(ctx context.Context, userID, statusFilter string) ([]model.Consultation, error) {
	return m.getByUserFn(ctx, userID, statusFilter)
}
func (m *mockConsultationRepo) Complete(ctx context.Context, id, notes string) error {
	return m.completeFn(ctx, id, notes)
}

type mockModificationRepo struct {
	createFn    func(ctx context.Context, tm *model.TrainingModification) error
	getByUserFn func(ctx context.Context, userID, trainingPlanID string) ([]model.TrainingModification, error)
}

func (m *mockModificationRepo) Create(ctx context.Context, tm *model.TrainingModification) error {
	return m.createFn(ctx, tm)
}
func (m *mockModificationRepo) GetByUser(ctx context.Context, userID, trainingPlanID string) ([]model.TrainingModification, error) {
	return m.getByUserFn(ctx, userID, trainingPlanID)
}

func newTestService(
	doctors port.DoctorRepository,
	subscriptions port.SubscriptionRepository,
	messages port.MessageRepository,
	prescriptions port.PrescriptionRepository,
	consultations port.ConsultationRepository,
	modifications port.TrainingModificationRepository,
) *Service {
	return New(doctors, subscriptions, messages, prescriptions, consultations, modifications)
}

// ===== Tests =====

func TestNewService(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)
	assert.NotNil(t, svc)
}

// --- Doctors ---

func TestListDoctors(t *testing.T) {
	expectedDoctors := []model.Doctor{{ID: "d1", Specialty: "sports_medicine"}}
	repo := &mockDoctorRepo{
		listFn: func(ctx context.Context, specialty string, page, pageSize int32) ([]model.Doctor, int64, error) {
			return expectedDoctors, 1, nil
		},
	}
	svc := newTestService(repo, nil, nil, nil, nil, nil)

	doctors, total, err := svc.ListDoctors(context.Background(), "sports_medicine", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, expectedDoctors, doctors)
	assert.Equal(t, int64(1), total)
}

func TestListDoctorsError(t *testing.T) {
	repo := &mockDoctorRepo{
		listFn: func(ctx context.Context, specialty string, page, pageSize int32) ([]model.Doctor, int64, error) {
			return nil, 0, errors.New("db error")
		},
	}
	svc := newTestService(repo, nil, nil, nil, nil, nil)

	doctors, total, err := svc.ListDoctors(context.Background(), "", 1, 10)
	assert.Error(t, err)
	assert.Nil(t, doctors)
	assert.Equal(t, int64(0), total)
}

func TestGetDoctor(t *testing.T) {
	expected := &model.Doctor{ID: "d1", Specialty: "general_practice"}
	repo := &mockDoctorRepo{
		getByIDFn: func(ctx context.Context, id string) (*model.Doctor, error) {
			return expected, nil
		},
	}
	svc := newTestService(repo, nil, nil, nil, nil, nil)

	doctor, err := svc.GetDoctor(context.Background(), "d1")
	require.NoError(t, err)
	assert.Equal(t, expected, doctor)
}

func TestGetDoctorError(t *testing.T) {
	repo := &mockDoctorRepo{
		getByIDFn: func(ctx context.Context, id string) (*model.Doctor, error) {
			return nil, errors.New("not found")
		},
	}
	svc := newTestService(repo, nil, nil, nil, nil, nil)

	doctor, err := svc.GetDoctor(context.Background(), "unknown")
	assert.Error(t, err)
	assert.Nil(t, doctor)
}

// --- Subscriptions ---

func TestSubscribeMonthly(t *testing.T) {
	var created *model.Subscription
	subRepo := &mockSubscriptionRepo{
		createFn: func(ctx context.Context, s *model.Subscription) error {
			created = s
			return nil
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	sub, err := svc.Subscribe(context.Background(), "user-1", "doc-1", "monthly")
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, "user-1", sub.UserID)
	assert.Equal(t, "doc-1", sub.DoctorID)
	assert.Equal(t, "monthly", sub.PlanType)
	assert.True(t, sub.IsActive)
	assert.Equal(t, 999.0, sub.Price)
	assert.Equal(t, created, sub)
}

func TestSubscribeQuarterly(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		createFn: func(ctx context.Context, s *model.Subscription) error { return nil },
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	sub, err := svc.Subscribe(context.Background(), "u1", "d1", "quarterly")
	require.NoError(t, err)
	assert.Equal(t, 2499.0, sub.Price)
	assert.True(t, sub.ExpiresAt.After(sub.StartsAt))
}

func TestSubscribeYearly(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		createFn: func(ctx context.Context, s *model.Subscription) error { return nil },
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	sub, err := svc.Subscribe(context.Background(), "u1", "d1", "yearly")
	require.NoError(t, err)
	assert.Equal(t, 7999.0, sub.Price)
}

func TestSubscribeInvalidPlan(t *testing.T) {
	subRepo := &mockSubscriptionRepo{}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	sub, err := svc.Subscribe(context.Background(), "u1", "d1", "invalid_plan")
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Contains(t, err.Error(), "invalid plan type")
}

func TestSubscribeError(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		createFn: func(ctx context.Context, s *model.Subscription) error {
			return errors.New("db error")
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	sub, err := svc.Subscribe(context.Background(), "u1", "d1", "monthly")
	assert.Error(t, err)
	assert.Nil(t, sub)
}

func TestGetSubscription(t *testing.T) {
	expected := &model.Subscription{ID: "sub-1", UserID: "u1", DoctorID: "d1"}
	subRepo := &mockSubscriptionRepo{
		getByUserAndDoctorFn: func(ctx context.Context, userID, doctorID string) (*model.Subscription, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	sub, err := svc.GetSubscription(context.Background(), "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, expected, sub)
}

func TestCancelSubscription(t *testing.T) {
	called := false
	subRepo := &mockSubscriptionRepo{
		cancelFn: func(ctx context.Context, userID, doctorID string) error {
			called = true
			return nil
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	err := svc.CancelSubscription(context.Background(), "u1", "d1")
	require.NoError(t, err)
	assert.True(t, called)
}

// --- Messages ---

func TestSendMessageSuccess(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		isActiveFn: func(ctx context.Context, userID, doctorID string) (bool, error) {
			return true, nil
		},
	}
	msgRepo := &mockMessageRepo{
		createFn: func(ctx context.Context, m *model.Message) (*model.Message, error) {
			return m, nil
		},
	}
	svc := newTestService(nil, subRepo, msgRepo, nil, nil, nil)

	msg := &model.Message{UserID: "u1", DoctorID: "d1", Message: "Hello", MessageType: "text"}
	result, err := svc.SendMessage(context.Background(), msg)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.False(t, result.IsRead)
	assert.Equal(t, "Hello", result.Message)
}

func TestSendMessageNoSubscription(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		isActiveFn: func(ctx context.Context, userID, doctorID string) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	msg := &model.Message{UserID: "u1", DoctorID: "d1", Message: "Hello"}
	result, err := svc.SendMessage(context.Background(), msg)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "active subscription required")
}

func TestSendMessageSubscriptionError(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		isActiveFn: func(ctx context.Context, userID, doctorID string) (bool, error) {
			return false, errors.New("db error")
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	msg := &model.Message{UserID: "u1", DoctorID: "d1", Message: "Hello"}
	result, err := svc.SendMessage(context.Background(), msg)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestGetChatHistory(t *testing.T) {
	expected := []model.Message{{ID: "m1", Message: "Hello"}}
	msgRepo := &mockMessageRepo{
		getChatHistoryFn: func(ctx context.Context, userID, doctorID string, page, pageSize int32) ([]model.Message, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, nil, msgRepo, nil, nil, nil)

	msgs, err := svc.GetChatHistory(context.Background(), "u1", "d1", 1, 10)
	require.NoError(t, err)
	assert.Equal(t, expected, msgs)
}

func TestMarkMessagesRead(t *testing.T) {
	msgRepo := &mockMessageRepo{
		markAsReadFn: func(ctx context.Context, userID, doctorID string) (int64, error) {
			return 5, nil
		},
	}
	svc := newTestService(nil, nil, msgRepo, nil, nil, nil)

	count, err := svc.MarkMessagesRead(context.Background(), "u1", "d1")
	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
}

func TestGetUnreadCount(t *testing.T) {
	expected := map[string]int64{"d1": 3}
	msgRepo := &mockMessageRepo{
		getUnreadCountFn: func(ctx context.Context, userID string) (map[string]int64, int64, error) {
			return expected, 3, nil
		},
	}
	svc := newTestService(nil, nil, msgRepo, nil, nil, nil)

	counts, total, err := svc.GetUnreadCount(context.Background(), "u1")
	require.NoError(t, err)
	assert.Equal(t, expected, counts)
	assert.Equal(t, int64(3), total)
}

// --- Prescriptions ---

func TestCreatePrescription(t *testing.T) {
	rxRepo := &mockPrescriptionRepo{
		createFn: func(ctx context.Context, p *model.Prescription) (*model.Prescription, error) {
			return p, nil
		},
	}
	svc := newTestService(nil, nil, nil, rxRepo, nil, nil)

	p := &model.Prescription{
		UserID:           "u1",
		DoctorID:         "d1",
		PrescriptionType: "medication",
		Title:            "Test",
	}
	result, err := svc.CreatePrescription(context.Background(), p)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "active", result.Status)
	assert.False(t, result.CreatedAt.IsZero())
	assert.False(t, result.UpdatedAt.IsZero())
}

func TestGetPrescriptions(t *testing.T) {
	expected := []model.Prescription{{ID: "rx1"}}
	rxRepo := &mockPrescriptionRepo{
		getByUserFn: func(ctx context.Context, userID, statusFilter string) ([]model.Prescription, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, nil, nil, rxRepo, nil, nil)

	results, err := svc.GetPrescriptions(context.Background(), "u1", "active")
	require.NoError(t, err)
	assert.Equal(t, expected, results)
}

func TestUpdatePrescriptionStatus(t *testing.T) {
	called := false
	rxRepo := &mockPrescriptionRepo{
		updateStatusFn: func(ctx context.Context, id, status string) error {
			called = true
			assert.Equal(t, "rx1", id)
			assert.Equal(t, "completed", status)
			return nil
		},
	}
	svc := newTestService(nil, nil, nil, rxRepo, nil, nil)

	err := svc.UpdatePrescriptionStatus(context.Background(), "rx1", "completed")
	require.NoError(t, err)
	assert.True(t, called)
}

// --- Training Modifications ---

func TestCreateTrainingModification(t *testing.T) {
	modRepo := &mockModificationRepo{
		createFn: func(ctx context.Context, tm *model.TrainingModification) error {
			return nil
		},
	}
	svc := newTestService(nil, nil, nil, nil, nil, modRepo)

	tm := &model.TrainingModification{
		DoctorID:         "d1",
		TrainingPlanID:   "p1",
		ModificationType: "exercise_added",
	}
	err := svc.CreateTrainingModification(context.Background(), tm)
	require.NoError(t, err)
	assert.NotEmpty(t, tm.ID)
	assert.False(t, tm.CreatedAt.IsZero())
}

func TestGetTrainingModifications(t *testing.T) {
	expected := []model.TrainingModification{{ID: "tm1", ModificationType: "exercise_added"}}
	modRepo := &mockModificationRepo{
		getByUserFn: func(ctx context.Context, userID, trainingPlanID string) ([]model.TrainingModification, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, nil, nil, nil, nil, modRepo)

	results, err := svc.GetTrainingModifications(context.Background(), "u1", "p1")
	require.NoError(t, err)
	assert.Equal(t, expected, results)
}

// --- Consultations ---

func TestScheduleConsultationSuccess(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		isActiveFn: func(ctx context.Context, userID, doctorID string) (bool, error) {
			return true, nil
		},
	}
	conRepo := &mockConsultationRepo{
		createFn: func(ctx context.Context, c *model.Consultation) (*model.Consultation, error) {
			return c, nil
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, conRepo, nil)

	c := &model.Consultation{UserID: "u1", DoctorID: "d1", ScheduledAt: time.Now().Add(24 * time.Hour)}
	result, err := svc.ScheduleConsultation(context.Background(), c)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "scheduled", result.Status)
	assert.False(t, result.CreatedAt.IsZero())
}

func TestScheduleConsultationNoSubscription(t *testing.T) {
	subRepo := &mockSubscriptionRepo{
		isActiveFn: func(ctx context.Context, userID, doctorID string) (bool, error) {
			return false, nil
		},
	}
	svc := newTestService(nil, subRepo, nil, nil, nil, nil)

	c := &model.Consultation{UserID: "u1", DoctorID: "d1"}
	result, err := svc.ScheduleConsultation(context.Background(), c)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "active subscription required")
}

func TestGetConsultations(t *testing.T) {
	expected := []model.Consultation{{ID: "c1", Status: "scheduled"}}
	conRepo := &mockConsultationRepo{
		getByUserFn: func(ctx context.Context, userID, statusFilter string) ([]model.Consultation, error) {
			return expected, nil
		},
	}
	svc := newTestService(nil, nil, nil, nil, conRepo, nil)

	results, err := svc.GetConsultations(context.Background(), "u1", "scheduled")
	require.NoError(t, err)
	assert.Equal(t, expected, results)
}

func TestCompleteConsultation(t *testing.T) {
	called := false
	conRepo := &mockConsultationRepo{
		completeFn: func(ctx context.Context, id, notes string) error {
			called = true
			assert.Equal(t, "c1", id)
			assert.Equal(t, "Patient improved", notes)
			return nil
		},
	}
	svc := newTestService(nil, nil, nil, nil, conRepo, nil)

	err := svc.CompleteConsultation(context.Background(), "c1", "Patient improved")
	require.NoError(t, err)
	assert.True(t, called)
}
