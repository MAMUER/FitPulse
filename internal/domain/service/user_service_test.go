package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type mockUserRepository struct {
	createFn        func(ctx context.Context, user *entity.User) error
	getByIDFn       func(ctx context.Context, id string) (*entity.User, error)
	getByEmailFn    func(ctx context.Context, email string) (*entity.User, error)
	updateFn        func(ctx context.Context, user *entity.User) error
	deleteFn        func(ctx context.Context, id string) error
	listFn          func(ctx context.Context, page, pageSize int) ([]*entity.User, error)
	countFn         func(ctx context.Context) (int, error)
	existsByEmailFn func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserRepository) Create(ctx context.Context, user *entity.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, apperrors.NotFound("not found")
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, apperrors.NotFound("not found")
}

func (m *mockUserRepository) Update(ctx context.Context, user *entity.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, page, pageSize int) ([]*entity.User, error) {
	if m.listFn != nil {
		return m.listFn(ctx, page, pageSize)
	}
	return nil, nil
}

func (m *mockUserRepository) Count(ctx context.Context) (int, error) {
	if m.countFn != nil {
		return m.countFn(ctx)
	}
	return 0, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}

func (m *mockUserRepository) ListByRole(ctx context.Context, role string, page, pageSize int) ([]*entity.User, int, error) {
	return nil, 0, nil
}

var _ port.UserRepository = (*mockUserRepository)(nil)

type mockProfileRepository struct {
	getProfileFn    func(ctx context.Context, userID string) (*entity.User, error)
	updateProfileFn func(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error
	userExistsFn    func(ctx context.Context, userID string) (bool, error)
	createProfileFn func(ctx context.Context, userID string) error
	upsertProfileFn func(ctx context.Context, userID string, data *port.ProfileData) error
}

func (m *mockProfileRepository) GetProfile(ctx context.Context, userID string) (*entity.User, error) {
	if m.getProfileFn != nil {
		return m.getProfileFn(ctx, userID)
	}
	return &entity.User{ID: userID}, nil
}

func (m *mockProfileRepository) UpdateProfile(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error {
	if m.updateProfileFn != nil {
		return m.updateProfileFn(ctx, userID, fullName, goals, contraindications, nutrition, sleepHours)
	}
	return nil
}

func (m *mockProfileRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	if m.userExistsFn != nil {
		return m.userExistsFn(ctx, userID)
	}
	return true, nil
}

func (m *mockProfileRepository) CreateProfile(ctx context.Context, userID string) error {
	if m.createProfileFn != nil {
		return m.createProfileFn(ctx, userID)
	}
	return nil
}

func (m *mockProfileRepository) UpsertProfile(ctx context.Context, userID string, data *port.ProfileData) error {
	if m.upsertProfileFn != nil {
		return m.upsertProfileFn(ctx, userID, data)
	}
	return nil
}

var _ port.ProfileRepository = (*mockProfileRepository)(nil)

type mockInviteRepository struct {
	createFn    func(ctx context.Context, invite *port.Invite) error
	getByCodeFn func(ctx context.Context, code string) (*port.Invite, error)
	listFn      func(ctx context.Context, page, pageSize int) ([]*port.Invite, int, error)
	revokeFn    func(ctx context.Context, code string) error
}

func (m *mockInviteRepository) Create(ctx context.Context, invite *port.Invite) error {
	if m.createFn != nil {
		return m.createFn(ctx, invite)
	}
	return nil
}

func (m *mockInviteRepository) GetByCode(ctx context.Context, code string) (*port.Invite, error) {
	if m.getByCodeFn != nil {
		return m.getByCodeFn(ctx, code)
	}
	return &port.Invite{Code: code, IsActive: true, UsedCount: 0, MaxUses: 1}, nil
}

func (m *mockInviteRepository) List(ctx context.Context, page, pageSize int) ([]*port.Invite, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockInviteRepository) Revoke(ctx context.Context, code string) error {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, code)
	}
	return nil
}

var _ port.InviteRepository = (*mockInviteRepository)(nil)

type mockInviteCodeRepository struct {
	listFn                  func(ctx context.Context, page, pageSize int) ([]*port.InviteCode, int, error)
	createFn                func(ctx context.Context, invite *port.InviteCode) error
	revokeFn                func(ctx context.Context, code string) error
	validateFn              func(ctx context.Context, code string) (*port.InviteCode, error)
	useInviteCodeFn         func(ctx context.Context, code string) error
	validateInviteCodeUseFn func(ctx context.Context, code string) (bool, string, string, string, error)
	logInviteCodeUseFn      func(ctx context.Context, code, userID string) error
}

func (m *mockInviteCodeRepository) List(ctx context.Context, page, pageSize int) ([]*port.InviteCode, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, page, pageSize)
	}
	return nil, 0, nil
}

func (m *mockInviteCodeRepository) Create(ctx context.Context, invite *port.InviteCode) error {
	if m.createFn != nil {
		return m.createFn(ctx, invite)
	}
	return nil
}

func (m *mockInviteCodeRepository) Revoke(ctx context.Context, code string) error {
	if m.revokeFn != nil {
		return m.revokeFn(ctx, code)
	}
	return nil
}

func (m *mockInviteCodeRepository) Validate(ctx context.Context, code string) (*port.InviteCode, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, code)
	}
	return nil, nil
}

func (m *mockInviteCodeRepository) UseInviteCode(ctx context.Context, code string) error {
	if m.useInviteCodeFn != nil {
		return m.useInviteCodeFn(ctx, code)
	}
	return nil
}

func (m *mockInviteCodeRepository) ValidateInviteCodeUse(ctx context.Context, code string) (bool, string, string, string, error) {
	if m.validateInviteCodeUseFn != nil {
		return m.validateInviteCodeUseFn(ctx, code)
	}
	return false, "", "", "", nil
}

func (m *mockInviteCodeRepository) LogInviteCodeUse(ctx context.Context, code, userID string) error {
	if m.logInviteCodeUseFn != nil {
		return m.logInviteCodeUseFn(ctx, code, userID)
	}
	return nil
}

var _ port.InviteCodeRepository = (*mockInviteCodeRepository)(nil)

type mockHealthConditionRepository struct {
	createFn func(ctx context.Context, condition *entity.HealthCondition) (*entity.HealthCondition, error)
	listFn   func(ctx context.Context, userID, conditionType string) ([]*entity.HealthCondition, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockHealthConditionRepository) Create(ctx context.Context, condition *entity.HealthCondition) (*entity.HealthCondition, error) {
	if m.createFn != nil {
		return m.createFn(ctx, condition)
	}
	return condition, nil
}

func (m *mockHealthConditionRepository) List(ctx context.Context, userID, conditionType string) ([]*entity.HealthCondition, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, conditionType)
	}
	return nil, nil
}

func (m *mockHealthConditionRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

var _ port.HealthConditionRepository = (*mockHealthConditionRepository)(nil)

type mockUserHealthConditionRepository struct {
	listFn   func(ctx context.Context, userID string) ([]*port.UserHealthCondition, error)
	upsertFn func(ctx context.Context, condition *port.UserHealthCondition) (*port.UserHealthCondition, error)
	deleteFn func(ctx context.Context, id, userID string) error
}

func (m *mockUserHealthConditionRepository) List(ctx context.Context, userID string) ([]*port.UserHealthCondition, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserHealthConditionRepository) Upsert(ctx context.Context, condition *port.UserHealthCondition) (*port.UserHealthCondition, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, condition)
	}
	return condition, nil
}

func (m *mockUserHealthConditionRepository) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, userID)
	}
	return nil
}

var _ port.UserHealthConditionRepository = (*mockUserHealthConditionRepository)(nil)

type mockBodyCompositionRepository struct {
	createFn func(ctx context.Context, bc *entity.BodyComposition) (*entity.BodyComposition, error)
	listFn   func(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*entity.BodyComposition, error)
}

func (m *mockBodyCompositionRepository) Create(ctx context.Context, bc *entity.BodyComposition) (*entity.BodyComposition, error) {
	if m.createFn != nil {
		return m.createFn(ctx, bc)
	}
	return bc, nil
}

func (m *mockBodyCompositionRepository) List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*entity.BodyComposition, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, from, to, limit)
	}
	return nil, nil
}

var _ port.BodyCompositionRepository = (*mockBodyCompositionRepository)(nil)

type mockUserBodyCompositionRepository struct {
	listFn   func(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*port.UserBodyComposition, error)
	createFn func(ctx context.Context, bc *port.UserBodyComposition) (*port.UserBodyComposition, error)
}

func (m *mockUserBodyCompositionRepository) List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*port.UserBodyComposition, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, from, to, limit)
	}
	return nil, nil
}

func (m *mockUserBodyCompositionRepository) Create(ctx context.Context, bc *port.UserBodyComposition) (*port.UserBodyComposition, error) {
	if m.createFn != nil {
		return m.createFn(ctx, bc)
	}
	return bc, nil
}

var _ port.UserBodyCompositionRepository = (*mockUserBodyCompositionRepository)(nil)

type mockMenstrualCycleRepository struct {
	createFn func(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	listFn   func(ctx context.Context, userID string) ([]*entity.MenstrualCycle, error)
	updateFn func(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	deleteFn func(ctx context.Context, id string) error
}

func (m *mockMenstrualCycleRepository) Create(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error) {
	if m.createFn != nil {
		return m.createFn(ctx, cycle)
	}
	return cycle, nil
}

func (m *mockMenstrualCycleRepository) List(ctx context.Context, userID string) ([]*entity.MenstrualCycle, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockMenstrualCycleRepository) Update(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, cycle)
	}
	return cycle, nil
}

func (m *mockMenstrualCycleRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

var _ port.MenstrualCycleRepository = (*mockMenstrualCycleRepository)(nil)

type mockUserMenstrualRepository struct {
	listCyclesFn             func(ctx context.Context, userID string) ([]*port.UserMenstrualCycle, error)
	createCycleFn            func(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error)
	createCycleWithDetailsFn func(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error)
	updateCycleFn            func(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error)
	updateCycleWithDetailsFn func(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error)
	deleteCycleFn            func(ctx context.Context, id, userID string) error
	listSymptomsFn           func(ctx context.Context, cycleID string) ([]string, error)
	createSymptomFn          func(ctx context.Context, cycleID, symptom string) error
	deleteSymptomsFn         func(ctx context.Context, cycleID string) error
	listMoodsFn              func(ctx context.Context, cycleID string) ([]string, error)
	createMoodFn             func(ctx context.Context, cycleID, mood string) error
	deleteMoodsFn            func(ctx context.Context, cycleID string) error
}

func (m *mockUserMenstrualRepository) ListCycles(ctx context.Context, userID string) ([]*port.UserMenstrualCycle, error) {
	if m.listCyclesFn != nil {
		return m.listCyclesFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockUserMenstrualRepository) CreateCycle(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	if m.createCycleFn != nil {
		return m.createCycleFn(ctx, cycle)
	}
	return cycle, nil
}

func (m *mockUserMenstrualRepository) CreateCycleWithDetails(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	if m.createCycleWithDetailsFn != nil {
		return m.createCycleWithDetailsFn(ctx, cycle)
	}
	return cycle, nil
}

func (m *mockUserMenstrualRepository) UpdateCycle(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	if m.updateCycleFn != nil {
		return m.updateCycleFn(ctx, cycle)
	}
	return cycle, nil
}

func (m *mockUserMenstrualRepository) UpdateCycleWithDetails(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	if m.updateCycleWithDetailsFn != nil {
		return m.updateCycleWithDetailsFn(ctx, cycle)
	}
	return cycle, nil
}

func (m *mockUserMenstrualRepository) DeleteCycle(ctx context.Context, id, userID string) error {
	if m.deleteCycleFn != nil {
		return m.deleteCycleFn(ctx, id, userID)
	}
	return nil
}

func (m *mockUserMenstrualRepository) ListSymptoms(ctx context.Context, cycleID string) ([]string, error) {
	if m.listSymptomsFn != nil {
		return m.listSymptomsFn(ctx, cycleID)
	}
	return nil, nil
}

func (m *mockUserMenstrualRepository) CreateSymptom(ctx context.Context, cycleID, symptom string) error {
	if m.createSymptomFn != nil {
		return m.createSymptomFn(ctx, cycleID, symptom)
	}
	return nil
}

func (m *mockUserMenstrualRepository) DeleteSymptoms(ctx context.Context, cycleID string) error {
	if m.deleteSymptomsFn != nil {
		return m.deleteSymptomsFn(ctx, cycleID)
	}
	return nil
}

func (m *mockUserMenstrualRepository) ListMoods(ctx context.Context, cycleID string) ([]string, error) {
	if m.listMoodsFn != nil {
		return m.listMoodsFn(ctx, cycleID)
	}
	return nil, nil
}

func (m *mockUserMenstrualRepository) CreateMood(ctx context.Context, cycleID, mood string) error {
	if m.createMoodFn != nil {
		return m.createMoodFn(ctx, cycleID, mood)
	}
	return nil
}

func (m *mockUserMenstrualRepository) DeleteMoods(ctx context.Context, cycleID string) error {
	if m.deleteMoodsFn != nil {
		return m.deleteMoodsFn(ctx, cycleID)
	}
	return nil
}

var _ port.UserMenstrualRepository = (*mockUserMenstrualRepository)(nil)

type mockAchievementRepository struct {
	listFn func(ctx context.Context, userID string) ([]*entity.Achievement, error)
}

func (m *mockAchievementRepository) Create(ctx context.Context, achievement *entity.Achievement) (*entity.Achievement, error) {
	if m.listFn != nil {
		return achievement, nil
	}
	return achievement, nil
}

func (m *mockAchievementRepository) List(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	return nil, nil
}

var _ port.AchievementRepository = (*mockAchievementRepository)(nil)

type mockAchievementRepositoryEx struct {
	listWithEarnedStatusFn func(ctx context.Context, userID string) ([]*port.AchievementInfo, error)
	earnFn                 func(ctx context.Context, userID, achievementID string) error
}

func (m *mockAchievementRepositoryEx) ListWithEarnedStatus(ctx context.Context, userID string) ([]*port.AchievementInfo, error) {
	if m.listWithEarnedStatusFn != nil {
		return m.listWithEarnedStatusFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockAchievementRepositoryEx) Earn(ctx context.Context, userID, achievementID string) error {
	if m.earnFn != nil {
		return m.earnFn(ctx, userID, achievementID)
	}
	return nil
}

var _ port.AchievementRepositoryEx = (*mockAchievementRepositoryEx)(nil)

type mockDeviceRepository struct {
	listFn   func(ctx context.Context, userID string) ([]*entity.Device, error)
	createFn func(ctx context.Context, device *entity.Device) (*entity.Device, error)
	deleteFn func(ctx context.Context, userID, deviceID string) error
}

func (m *mockDeviceRepository) List(ctx context.Context, userID string) ([]*entity.Device, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockDeviceRepository) Create(ctx context.Context, device *entity.Device) (*entity.Device, error) {
	if m.createFn != nil {
		return m.createFn(ctx, device)
	}
	return device, nil
}

func (m *mockDeviceRepository) Delete(ctx context.Context, userID, deviceID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, deviceID)
	}
	return nil
}

var _ port.DeviceRepository = (*mockDeviceRepository)(nil)

type mockEmailVerificationRepository struct {
	createFn        func(ctx context.Context, ev *port.EmailVerification) error
	getValidTokenFn func(ctx context.Context, token string) (*port.EmailVerification, error)
	getByUserIDFn   func(ctx context.Context, userID string) (*port.EmailVerification, error)
	markUsedFn      func(ctx context.Context, token string) error
	markVerifiedFn  func(ctx context.Context, userID string) error
}

func (m *mockEmailVerificationRepository) Create(ctx context.Context, ev *port.EmailVerification) error {
	if m.createFn != nil {
		return m.createFn(ctx, ev)
	}
	return nil
}

func (m *mockEmailVerificationRepository) GetValidToken(ctx context.Context, token string) (*port.EmailVerification, error) {
	if m.getValidTokenFn != nil {
		return m.getValidTokenFn(ctx, token)
	}
	return nil, nil
}

func (m *mockEmailVerificationRepository) GetByUserID(ctx context.Context, userID string) (*port.EmailVerification, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockEmailVerificationRepository) MarkUsed(ctx context.Context, token string) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(ctx, token)
	}
	return nil
}

func (m *mockEmailVerificationRepository) MarkUserEmailVerified(ctx context.Context, userID string) error {
	if m.markVerifiedFn != nil {
		return m.markVerifiedFn(ctx, userID)
	}
	return nil
}

var _ port.EmailVerificationRepository = (*mockEmailVerificationRepository)(nil)

type mockRefreshTokenRepository struct {
	getValidFn func(ctx context.Context, token string) (*port.RefreshToken, error)
	createFn   func(ctx context.Context, rt *port.RefreshToken) error
	markUsedFn func(ctx context.Context, token string) error
}

func (m *mockRefreshTokenRepository) GetValid(ctx context.Context, token string) (*port.RefreshToken, error) {
	if m.getValidFn != nil {
		return m.getValidFn(ctx, token)
	}
	return nil, nil
}

func (m *mockRefreshTokenRepository) Create(ctx context.Context, rt *port.RefreshToken) error {
	if m.createFn != nil {
		return m.createFn(ctx, rt)
	}
	return nil
}

func (m *mockRefreshTokenRepository) MarkUsed(ctx context.Context, token string) error {
	if m.markUsedFn != nil {
		return m.markUsedFn(ctx, token)
	}
	return nil
}

var _ port.RefreshTokenRepository = (*mockRefreshTokenRepository)(nil)

func TestRegister(t *testing.T) {
	t.Run("returns validation error when email is empty", func(t *testing.T) {
		mockUsers := &mockUserRepository{}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, err := svc.Register(context.Background(), "", "password", "John", "user")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when password is empty", func(t *testing.T) {
		mockUsers := &mockUserRepository{}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, err := svc.Register(context.Background(), "john@example.com", "", "John", "user")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns conflict when email already exists", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
				return true, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, err := svc.Register(context.Background(), "john@example.com", "password", "John", "user")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "CONFLICT", apperrors.Code(err))
	})

	t.Run("returns repository error on exists check", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
				return false, apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, err := svc.Register(context.Background(), "john@example.com", "password", "John", "user")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})

	t.Run("returns repository error on create", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, _ *entity.User) error {
				return apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, err := svc.Register(context.Background(), "john@example.com", "password", "John", "user")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})

	t.Run("creates user and returns it", func(t *testing.T) {
		var capturedUser *entity.User
		mockUsers := &mockUserRepository{
			existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
				return false, nil
			},
			createFn: func(_ context.Context, u *entity.User) error {
				capturedUser = u
				return nil
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, err := svc.Register(context.Background(), "john@example.com", "password", "John", "user")

		require.NoError(t, err)
		assert.Equal(t, "john@example.com", user.Email)
		assert.Equal(t, "John", user.FullName)
		assert.Equal(t, "user", user.Role)
		assert.False(t, user.EmailVerified)
		assert.NotEmpty(t, user.ID)
		assert.Equal(t, "john@example.com", capturedUser.Email)
		assert.False(t, capturedUser.CreatedAt.IsZero())
		assert.False(t, capturedUser.UpdatedAt.IsZero())
	})
}

func TestLogin(t *testing.T) {
	t.Run("returns validation error when email is empty", func(t *testing.T) {
		mockUsers := &mockUserRepository{}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, token, err := svc.Login(context.Background(), "", "password")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when password is empty", func(t *testing.T) {
		mockUsers := &mockUserRepository{}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, token, err := svc.Login(context.Background(), "john@example.com", "")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns unauthorized when user not found", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			getByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
				return nil, apperrors.ErrNotFound
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, token, err := svc.Login(context.Background(), "john@example.com", "password")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, "UNAUTHORIZED", apperrors.Code(err))
	})

	t.Run("returns other repository errors as-is", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			getByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, token, err := svc.Login(context.Background(), "john@example.com", "password")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Empty(t, token)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})

	t.Run("returns user when found", func(t *testing.T) {
		expectedUser := &entity.User{ID: "u1", Email: "john@example.com"}
		mockUsers := &mockUserRepository{
			getByEmailFn: func(_ context.Context, _ string) (*entity.User, error) {
				return expectedUser, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		user, token, err := svc.Login(context.Background(), "john@example.com", "password")

		require.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		assert.Empty(t, token)
	})
}

func TestGetProfile(t *testing.T) {
	t.Run("delegates to profile repository", func(t *testing.T) {
		expected := &entity.User{ID: "user1"}
		mockProfiles := &mockProfileRepository{
			getProfileFn: func(_ context.Context, _ string) (*entity.User, error) {
				return expected, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Profiles: mockProfiles})

		user, err := svc.GetProfile(context.Background(), "user1")

		require.NoError(t, err)
		assert.Equal(t, expected, user)
	})

	t.Run("returns profile repository error", func(t *testing.T) {
		mockProfiles := &mockProfileRepository{
			getProfileFn: func(_ context.Context, _ string) (*entity.User, error) {
				return nil, apperrors.NotFound("profile not found")
			},
		}
		svc := NewUserService(UserServiceConfig{Profiles: mockProfiles})

		user, err := svc.GetProfile(context.Background(), "user1")

		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "NOT_FOUND", apperrors.Code(err))
	})
}

func TestUpdateProfile(t *testing.T) {
	t.Run("delegates to profile repository", func(t *testing.T) {
		var capturedUserID string
		mockProfiles := &mockProfileRepository{
			updateProfileFn: func(_ context.Context, userID, _ string, _ []string, _ []string, _ string, _ float32) error {
				capturedUserID = userID
				return nil
			},
		}
		svc := NewUserService(UserServiceConfig{Profiles: mockProfiles})

		err := svc.UpdateProfile(context.Background(), "user1", "John", []string{"fitness"}, []string{"none"}, "balanced", 7.5)

		require.NoError(t, err)
		assert.Equal(t, "user1", capturedUserID)
	})

	t.Run("returns profile repository error", func(t *testing.T) {
		mockProfiles := &mockProfileRepository{
			updateProfileFn: func(_ context.Context, _ string, _ string, _ []string, _ []string, _ string, _ float32) error {
				return apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Profiles: mockProfiles})

		err := svc.UpdateProfile(context.Background(), "user1", "John", nil, nil, "", 0)

		require.Error(t, err)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}

func TestDeleteProfile(t *testing.T) {
	t.Run("returns error when user not found", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			getByIDFn: func(_ context.Context, _ string) (*entity.User, error) {
				return nil, apperrors.NotFound("user not found")
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		err := svc.DeleteProfile(context.Background(), "user1", "password")

		require.Error(t, err)
		assert.Equal(t, "NOT_FOUND", apperrors.Code(err))
	})

	t.Run("delegates delete to user repository", func(t *testing.T) {
		var capturedID string
		mockUsers := &mockUserRepository{
			getByIDFn: func(_ context.Context, id string) (*entity.User, error) {
				return &entity.User{ID: id}, nil
			},
			deleteFn: func(_ context.Context, id string) error {
				capturedID = id
				return nil
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		err := svc.DeleteProfile(context.Background(), "user1", "password")

		require.NoError(t, err)
		assert.Equal(t, "user1", capturedID)
	})
}

func TestConfirmEmail(t *testing.T) {
	t.Run("returns nil", func(t *testing.T) {
		svc := NewUserService(UserServiceConfig{})

		err := svc.ConfirmEmail(context.Background(), "token123")

		assert.NoError(t, err)
	})
}

func TestCreateInvite(t *testing.T) {
	t.Run("creates invite and returns code", func(t *testing.T) {
		var capturedCode string
		mockInvites := &mockInviteRepository{
			createFn: func(_ context.Context, invite *port.Invite) error {
				capturedCode = invite.Code
				return nil
			},
		}
		svc := NewUserService(UserServiceConfig{Invites: mockInvites})

		code, err := svc.CreateInvite(context.Background(), "admin", "cardiology", 10)

		require.NoError(t, err)
		assert.NotEmpty(t, code)
		assert.Equal(t, code, capturedCode)
		assert.Len(t, code, 12)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mockInvites := &mockInviteRepository{
			createFn: func(_ context.Context, _ *port.Invite) error {
				return apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Invites: mockInvites})

		code, err := svc.CreateInvite(context.Background(), "admin", "cardiology", 10)

		require.Error(t, err)
		assert.Empty(t, code)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}

func TestValidateInvite(t *testing.T) {
	t.Run("returns conflict when invite code is not active", func(t *testing.T) {
		mockInvites := &mockInviteRepository{
			getByCodeFn: func(_ context.Context, _ string) (*port.Invite, error) {
				return &port.Invite{Code: "code1", IsActive: false, UsedCount: 0, MaxUses: 1}, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Invites: mockInvites})

		role, specialty, err := svc.ValidateInvite(context.Background(), "code1")

		require.Error(t, err)
		assert.Empty(t, role)
		assert.Empty(t, specialty)
		assert.Equal(t, "CONFLICT", apperrors.Code(err))
	})

	t.Run("returns conflict when invite code exhausted", func(t *testing.T) {
		mockInvites := &mockInviteRepository{
			getByCodeFn: func(_ context.Context, _ string) (*port.Invite, error) {
				return &port.Invite{Code: "code1", IsActive: true, UsedCount: 1, MaxUses: 1}, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Invites: mockInvites})

		role, specialty, err := svc.ValidateInvite(context.Background(), "code1")

		require.Error(t, err)
		assert.Empty(t, role)
		assert.Empty(t, specialty)
		assert.Equal(t, "CONFLICT", apperrors.Code(err))
	})

	t.Run("returns role and specialty for valid invite", func(t *testing.T) {
		mockInvites := &mockInviteRepository{
			getByCodeFn: func(_ context.Context, _ string) (*port.Invite, error) {
				return &port.Invite{Code: "code1", IsActive: true, UsedCount: 0, MaxUses: 1, Role: "admin", Specialty: "cardiology"}, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Invites: mockInvites})

		role, specialty, err := svc.ValidateInvite(context.Background(), "code1")

		require.NoError(t, err)
		assert.Equal(t, "admin", role)
		assert.Equal(t, "cardiology", specialty)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mockInvites := &mockInviteRepository{
			getByCodeFn: func(_ context.Context, _ string) (*port.Invite, error) {
				return nil, apperrors.NotFound("invite not found")
			},
		}
		svc := NewUserService(UserServiceConfig{Invites: mockInvites})

		role, specialty, err := svc.ValidateInvite(context.Background(), "code1")

		require.Error(t, err)
		assert.Empty(t, role)
		assert.Empty(t, specialty)
		assert.Equal(t, "NOT_FOUND", apperrors.Code(err))
	})
}

func TestListUsers(t *testing.T) {
	t.Run("delegates to user repository and returns users with total", func(t *testing.T) {
		expectedUsers := []*entity.User{{ID: "u1"}, {ID: "u2"}}
		mockUsers := &mockUserRepository{
			listFn: func(_ context.Context, _ int, _ int) ([]*entity.User, error) {
				return expectedUsers, nil
			},
			countFn: func(_ context.Context) (int, error) {
				return 2, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		users, total, err := svc.ListUsers(context.Background(), 1, 20)

		require.NoError(t, err)
		assert.Equal(t, expectedUsers, users)
		assert.Equal(t, 2, total)
	})

	t.Run("returns list error", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			listFn: func(_ context.Context, _ int, _ int) ([]*entity.User, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		users, total, err := svc.ListUsers(context.Background(), 1, 20)

		require.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, 0, total)
	})

	t.Run("returns count error", func(t *testing.T) {
		mockUsers := &mockUserRepository{
			listFn: func(_ context.Context, _ int, _ int) ([]*entity.User, error) {
				return nil, nil
			},
			countFn: func(_ context.Context) (int, error) {
				return 0, apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Users: mockUsers})

		users, total, err := svc.ListUsers(context.Background(), 1, 20)

		require.Error(t, err)
		assert.Nil(t, users)
		assert.Equal(t, 0, total)
	})
}

func TestListDevices(t *testing.T) {
	t.Run("delegates to device repository", func(t *testing.T) {
		expected := []*entity.Device{{ID: "d1", DeviceType: "watch"}}
		mockDevices := &mockDeviceRepository{
			listFn: func(_ context.Context, _ string) ([]*entity.Device, error) {
				return expected, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		devices, err := svc.ListDevices(context.Background(), "user1")

		require.NoError(t, err)
		assert.Equal(t, expected, devices)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{
			listFn: func(_ context.Context, _ string) ([]*entity.Device, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		devices, err := svc.ListDevices(context.Background(), "user1")

		require.Error(t, err)
		assert.Nil(t, devices)
	})
}

func TestAddDevice(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		device := &entity.Device{DeviceType: "watch"}
		result, err := svc.AddDevice(context.Background(), device)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when device_type is empty", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		device := &entity.Device{UserID: "user1"}
		result, err := svc.AddDevice(context.Background(), device)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("sets default device name when empty", func(t *testing.T) {
		var capturedDevice *entity.Device
		mockDevices := &mockDeviceRepository{
			createFn: func(_ context.Context, d *entity.Device) (*entity.Device, error) {
				capturedDevice = d
				return d, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		device := &entity.Device{UserID: "user1", DeviceType: "watch"}
		_, err := svc.AddDevice(context.Background(), device)

		require.NoError(t, err)
		assert.Equal(t, "watch Device", capturedDevice.DeviceName)
		assert.True(t, capturedDevice.IsConnected)
		assert.False(t, capturedDevice.LastSync.IsZero())
	})

	t.Run("delegates create to repository", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{
			createFn: func(_ context.Context, d *entity.Device) (*entity.Device, error) {
				return d, nil
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		device := &entity.Device{UserID: "user1", DeviceType: "watch", DeviceName: "My Watch"}
		result, err := svc.AddDevice(context.Background(), device)

		require.NoError(t, err)
		assert.Equal(t, device, result)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{
			createFn: func(_ context.Context, _ *entity.Device) (*entity.Device, error) {
				return nil, apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		device := &entity.Device{UserID: "user1", DeviceType: "watch"}
		result, err := svc.AddDevice(context.Background(), device)

		require.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestRemoveDevice(t *testing.T) {
	t.Run("returns validation error when user_id is empty", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		err := svc.RemoveDevice(context.Background(), "", "d1")

		require.Error(t, err)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("returns validation error when device_id is empty", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		err := svc.RemoveDevice(context.Background(), "user1", "")

		require.Error(t, err)
		assert.Equal(t, "VALIDATION", apperrors.Code(err))
	})

	t.Run("delegates delete to repository", func(t *testing.T) {
		var capturedUserID, capturedDeviceID string
		mockDevices := &mockDeviceRepository{
			deleteFn: func(_ context.Context, userID, deviceID string) error {
				capturedUserID = userID
				capturedDeviceID = deviceID
				return nil
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		err := svc.RemoveDevice(context.Background(), "user1", "d1")

		require.NoError(t, err)
		assert.Equal(t, "user1", capturedUserID)
		assert.Equal(t, "d1", capturedDeviceID)
	})

	t.Run("returns repository error", func(t *testing.T) {
		mockDevices := &mockDeviceRepository{
			deleteFn: func(_ context.Context, _ string, _ string) error {
				return apperrors.Internal("db error", nil)
			},
		}
		svc := NewUserService(UserServiceConfig{Devices: mockDevices})

		err := svc.RemoveDevice(context.Background(), "user1", "d1")

		require.Error(t, err)
		assert.Equal(t, "INTERNAL", apperrors.Code(err))
	})
}
