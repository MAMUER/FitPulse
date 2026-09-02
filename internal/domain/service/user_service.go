package service

import (
	"context"
	"errors"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type userService struct {
	users        port.UserRepository
	profiles     port.ProfileRepository
	invites      port.InviteRepository
	health       port.HealthConditionRepository
	bodyComp     port.BodyCompositionRepository
	menstrual    port.MenstrualCycleRepository
	achievements port.AchievementRepository
}

type Invite struct {
	Code      string
	Role      string
	Specialty string
	MaxUses   int
	UsedCount int
	IsActive  bool
	CreatedAt time.Time
}

func NewUserService(
	users port.UserRepository,
	profiles port.ProfileRepository,
	invites port.InviteRepository,
	health port.HealthConditionRepository,
	bodyComp port.BodyCompositionRepository,
	menstrual port.MenstrualCycleRepository,
	achievements port.AchievementRepository,
) UserService {
	return &userService{
		users:        users,
		profiles:     profiles,
		invites:      invites,
		health:       health,
		bodyComp:     bodyComp,
		menstrual:    menstrual,
		achievements: achievements,
	}
}

func (s *userService) Register(ctx context.Context, email, password, fullName, role string) (*entity.User, error) {
	if email == "" || password == "" {
		return nil, apperrors.Validation("email and password are required")
	}

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.Conflict("user already exists")
	}

	user := &entity.User{
		ID:            generateID(),
		Email:         email,
		FullName:      fullName,
		Role:          role,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *userService) Login(ctx context.Context, email, password string) (*entity.User, string, error) {
	if email == "" || password == "" {
		return nil, "", apperrors.Validation("email and password are required")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, "", apperrors.Unauthorized("invalid credentials")
		}
		return nil, "", err
	}

	// Password verification should be done here
	// For now, return the user
	return user, "", nil
}

func (s *userService) GetProfile(ctx context.Context, userID string) (*entity.User, error) {
	user, err := s.profiles.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error {
	if err := s.profiles.UpdateProfile(ctx, userID, fullName, goals, contraindications, nutrition, sleepHours); err != nil {
		return err
	}
	return nil
}

func (s *userService) DeleteProfile(ctx context.Context, userID, password string) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Password verification should be done here
	_ = user

	return s.users.Delete(ctx, userID)
}

func (s *userService) ConfirmEmail(ctx context.Context, token string) error {
	// Email confirmation logic
	return nil
}

func (s *userService) CreateInvite(ctx context.Context, role, specialty string, maxUses int) (string, error) {
	code := generateInviteCode()
	invite := &Invite{
		Code:      code,
		Role:      role,
		Specialty: specialty,
		MaxUses:   maxUses,
		UsedCount: 0,
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	if err := s.invites.Create(ctx, invite); err != nil {
		return "", err
	}

	return code, nil
}

func (s *userService) ValidateInvite(ctx context.Context, code string) (string, string, error) {
	invite, err := s.invites.GetByCode(ctx, code)
	if err != nil {
		return "", "", err
	}

	if !invite.IsActive {
		return "", "", apperrors.Conflict("invite code is not active")
	}

	if invite.UsedCount >= invite.MaxUses {
		return "", "", apperrors.Conflict("invite code has been exhausted")
	}

	return invite.Role, invite.Specialty, nil
}

func (s *userService) ListUsers(ctx context.Context, page, pageSize int) ([]*entity.User, int, error) {
	users, err := s.users.List(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.users.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func generateInviteCode() string {
	return randomString(12)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
