package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/cmd/user-service/ports"
	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/audit"
	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/crypto"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
	"github.com/MAMUER/project/internal/domain/service"
	"github.com/MAMUER/project/internal/email"
	grpctls "github.com/MAMUER/project/internal/grpc"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/metrics"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/repository/postgres"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/MAMUER/project/internal/telemetry"
	"github.com/MAMUER/project/internal/totp"
	"github.com/MAMUER/project/internal/validator"
)

func intToInt32(v int) int32 {

	if v < math.MinInt32 || v > math.MaxInt32 {

		return 0

	}

	return int32(v)

}

// User represents a user for login operations.

type User struct {
	ID string

	Email string

	PasswordHash string

	Role string
}

type userServer struct {
	pb.UnimplementedUserServiceServer

	db *sql.DB

	userSvc service.UserService

	userRepo *postgres.PgsodiumUserRepository

	profileRepo port.ProfileRepository

	emailVerifRepo port.EmailVerificationRepository

	refreshTokenRepo port.RefreshTokenRepository

	achievementExRepo port.AchievementRepositoryEx

	inviteCodeRepo port.InviteCodeRepository

	userHealthRepo port.UserHealthConditionRepository

	userBodyCompRepo port.UserBodyCompositionRepository

	userMenstrualRepo port.UserMenstrualRepository

	deviceRepo port.DeviceRepository

	log *logger.Logger

	auditLogger *audit.ConsoleAuditLogger

	tokenProvider ports.TokenProvider

	emailSender email.EmailSender

	baseURL string

	googleClientID string

	totpService *totp.Service
}

const (
	argon2idParams = "m=65536,t=3,p=1"

	dateFormat = "2006-01-02"

	logFailedToGenerateNonce = "Failed to generate nonce"

	errFailedToGenerateNonce = "failed to generate nonce"

	logFailedToGenerateJWT = "Failed to generate JWT"

	errFailedToGenerateJWT = "failed to generate token"

	errInvalidCredentials = "invalid credentials"

	errUserNotFound = "user not found"

	errUserIDRequired = "user_id is required"

	errDatabaseError = "database error"

	errInternalError = "internal error"

	sqlValuesPrefix = "VALUES ($1, "

	sqlComma4Prefix = ", $4, "

	sql56Prefix = "$5, $6, "

	sqlSelectPrefix = "SELECT "

	sqlCommaNewlinePrefix = ",\n               "

	errFailedToValidateTOTP = "failed to validate TOTP code"

	errFailedToListDevices = "failed to list devices"
)

func hashPasswordArgon2id(password string) (string, error) {

	salt := make([]byte, 16)

	if _, err := rand.Read(salt); err != nil {

		return "", fmt.Errorf("generate salt: %w", err)

	}

	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, 32)

	return "$argon2id$v=19$" + argon2idParams + "$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash), nil

}

func verifyPasswordArgon2id(stored, password string) bool {

	parts := strings.Split(stored, "$")

	if len(parts) != 6 {

		return false

	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])

	if err != nil {

		return false

	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])

	if err != nil {

		return false

	}

	if len(hash) > 32 {

		return false

	}

	hashLen := len(hash)

	if uint64(hashLen) > uint64(^uint32(0)) {

		return false

	}

	computed := argon2.IDKey([]byte(password), salt, 3, 64*1024, 1, uint32(hashLen))

	return subtle.ConstantTimeCompare(hash, computed) == 1

}

func toString(v *string) string {

	if v == nil {

		return ""

	}

	return *v

}

func toInt32(v *int32) int32 {

	if v == nil {

		return 0

	}

	return *v

}

func toFloat64(v *float64) float64 {

	if v == nil {

		return 0

	}

	return *v

}

func toFloat32(v *float32) float32 {

	if v == nil {

		return 0

	}

	return *v

}

func float64Ptr(v float64) *float64 {

	return &v

}

func (s *userServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {

	s.log.Info("Register request", zap.String("email", req.Email))

	if err := validator.ValidateRegisterRequest(req); err != nil {

		s.log.Warn("Invalid register request", zap.Error(err))

		return nil, fmt.Errorf("validate register request: %w", err)

	}

	email := sanitize.String(req.Email)

	fullName := sanitize.String(req.FullName)

	emailHash := db.EmailHash(email)

	var exists bool

	exists, err := s.userRepo.ExistsByEmail(ctx, email)

	if err != nil {

		s.log.Error("Database error checking user existence", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if exists {

		return nil, status.Error(codes.AlreadyExists, "user already exists")

	}

	hashed, err := hashPasswordArgon2id(req.Password)

	if err != nil {

		s.log.Error("Failed to hash password", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to hash password")

	}

	userID := uuid.New().String()

	user := &entity.User{

		ID: userID,

		Email: email,

		PasswordHash: string(hashed),

		FullName: fullName,

		Role: req.Role,
	}

	err = s.userRepo.Create(ctx, user)

	if err != nil {

		s.log.Error("Failed to create user", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to create user")

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIAccess, "register", "user").
			WithUser(userID).
			WithFields("email", "full_name", "role").
			WithMetadata("email_hash", emailHash))

	}

	verificationToken := generateVerificationToken()

	emailVerificationNonce, err := db.GenerateNonce()

	if err != nil {

		s.log.Error(logFailedToGenerateNonce, zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToGenerateNonce)

	}

	tokenVerificationNonce, err := db.GenerateNonce()

	if err != nil {

		s.log.Error(logFailedToGenerateNonce, zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToGenerateNonce)

	}

	if err := s.userRepo.CreateEmailVerification(ctx, userID, email, emailHash, verificationToken, emailVerificationNonce, tokenVerificationNonce); err != nil {

		s.log.Error("Failed to create email verification record", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to create verification token")

	}

	sendVerificationEmailIfNeeded(ctx, s, email, verificationToken)

	if err := s.profileRepo.CreateProfile(ctx, userID); err != nil {

		s.log.Warn("Failed to create user profile, user will need to complete profile manually",

			zap.Error(err),

			zap.String("user_id", userID))

	}

	return &pb.RegisterResponse{

		UserId: userID,

		Message: "user created successfully. Please check your email to verify your account.",
	}, nil

}

func sendVerificationEmailIfNeeded(ctx context.Context, s *userServer, email, verificationToken string) {

	if s.emailSender == nil || s.baseURL == "" {

		return

	}

	if sendErr := s.emailSender.SendVerificationEmail(ctx, email, verificationToken, s.baseURL); sendErr != nil {

		s.log.Warn("Failed to send verification email (registration will proceed)",

			zap.Error(sendErr),

			zap.String("email", email))

	} else {

		s.log.Info("Verification email sent", zap.String("email", email))

	}

}

func (s *userServer) ConfirmEmail(ctx context.Context, req *pb.ConfirmEmailRequest) (*pb.ConfirmEmailResponse, error) {

	s.log.Info("Confirm email request", zap.String("token", req.Token))

	if req.Token == "" {

		return nil, status.Error(codes.InvalidArgument, "token is required")

	}

	userID, email, err := s.userRepo.ConfirmEmailWithPgsodium(ctx, req.Token)

	if err != nil {

		if apperrors.IsValidation(err) {

			return nil, status.Error(codes.InvalidArgument, err.Error())

		}

		s.log.Error("Database error checking verification token", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if err := s.emailVerifRepo.MarkUsed(ctx, req.Token); err != nil {

		s.log.Error("Failed to update verification token", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to confirm email")

	}

	if err := s.emailVerifRepo.MarkUserEmailVerified(ctx, userID); err != nil {

		s.log.Error("Failed to update user email_confirmed", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to confirm email")

	}

	s.log.Info("Email confirmed", zap.String("user_id", userID), zap.String("email", email))

	return &pb.ConfirmEmailResponse{

		UserId: userID,

		Message: "email confirmed successfully",
	}, nil

}

func (s *userServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {

	s.log.Info("Login request", zap.String("email", req.Email))

	if err := validator.ValidateLoginRequest(req); err != nil {

		s.log.Warn("Invalid login request", zap.Error(err))

		return nil, fmt.Errorf("validate login request: %w", err)

	}

	user, err := s.userRepo.LoginWithPgsodium(ctx, sanitize.String(req.Email))

	if err != nil {

		if apperrors.IsUnauthorized(err) {

			s.log.Info("Login attempt with unconfirmed email", zap.String("email", req.Email))

			return nil, status.Error(codes.Unauthenticated, "Email not confirmed. Please check your inbox.")

		}

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.Unauthenticated, errInvalidCredentials)

		}

		s.log.Error("Database error during login", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if !verifyPasswordArgon2id(user.PasswordHash, req.Password) {

		s.log.Info("Invalid login attempt", zap.String("email", req.Email))

		if s.auditLogger != nil {

			s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypeAuthFailure, "login", "user").
				WithUser(user.ID).
				WithFields("email").
				WithError(fmt.Errorf("invalid credentials")))

		}

		return nil, status.Error(codes.Unauthenticated, errInvalidCredentials)

	}

	token, err := s.tokenProvider.GenerateAccessToken(user.ID, user.Email, user.Role, 15*time.Minute)

	if err != nil {

		s.log.Error(logFailedToGenerateJWT, zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToGenerateJWT)

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypeAuthSuccess, "login", "user").
			WithUser(user.ID).
			WithFields("email"))

	}

	return &pb.LoginResponse{

		AccessToken: token,

		TokenType: "Bearer",

		ExpiresIn: 15 * 60,

		UserId: user.ID,

		Role: user.Role,
	}, nil

}

func (s *userServer) AuthenticateGoogle(ctx context.Context, req *pb.AuthenticateGoogleRequest) (*pb.LoginResponse, error) {

	s.log.Info("Google auth request")

	if req.IdToken == "" {

		return nil, status.Error(codes.InvalidArgument, "id_token is required")

	}

	payload, err := idtoken.Validate(ctx, req.IdToken, s.googleClientID)

	if err != nil {

		s.log.Warn("Invalid Google token", zap.Error(err))

		return nil, status.Error(codes.Unauthenticated, "invalid Google token")

	}

	emailVal, _ := payload.Claims["email"].(string)

	googleSub, _ := payload.Claims["sub"].(string)

	if emailVal == "" || googleSub == "" {

		return nil, status.Error(codes.InvalidArgument, "Google token missing required claims")

	}

	emailVal = sanitize.String(emailVal)

	emailHash := db.EmailHash(emailVal)

	userID, role, emailConfirmed, err := s.findOrCreateGoogleUser(ctx, googleSub, emailHash, emailVal)

	if err != nil {

		return nil, err

	}

	if !emailConfirmed {

		return nil, status.Error(codes.Unauthenticated, "email not confirmed")

	}

	token, tokenErr := s.tokenProvider.GenerateAccessToken(userID, emailVal, role, 15*time.Minute)

	if tokenErr != nil {

		s.log.Error(logFailedToGenerateJWT, zap.Error(tokenErr))

		return nil, status.Error(codes.Internal, errFailedToGenerateJWT)

	}

	s.log.Info("Google auth successful", zap.String("user_id", userID), zap.String("email", emailVal))

	return &pb.LoginResponse{

		AccessToken: token,

		TokenType: "Bearer",

		ExpiresIn: 15 * 60,

		UserId: userID,

		Role: role,
	}, nil

}

// Google OAuth flow requires multiple nonce generations and INSERT logic.

func (s *userServer) findOrCreateGoogleUser(ctx context.Context, googleSub, emailHash, emailVal string) (userID, role string, emailConfirmed bool, err error) {

	userID, role, emailConfirmed, err = s.findGoogleUserBySub(ctx, googleSub)

	if err == nil {

		return userID, role, emailConfirmed, nil

	}

	if !errors.Is(err, sql.ErrNoRows) {

		s.log.Error("Database error during Google auth", zap.Error(err))

		return "", "", false, status.Error(codes.Internal, errDatabaseError)

	}

	userID, role, emailConfirmed, err = s.linkGoogleToEmailUser(ctx, googleSub, emailHash)

	if err == nil {

		return userID, role, emailConfirmed, nil

	}

	if !errors.Is(err, sql.ErrNoRows) {

		s.log.Error("Database error during Google auth", zap.Error(err))

		return "", "", false, status.Error(codes.Internal, errDatabaseError)

	}

	return s.createGoogleUser(ctx, googleSub, emailHash, emailVal)

}

func (s *userServer) findGoogleUserBySub(ctx context.Context, googleSub string) (userID, role string, emailConfirmed bool, err error) {

	user, err := s.userRepo.FindGoogleUser(ctx, googleSub)

	if err == nil {

		if !user.EmailVerified {

			markErr := s.userRepo.MarkEmailVerified(ctx, user.ID)

			if markErr != nil {

				s.log.Warn("Failed to mark Google user email as confirmed", zap.Error(markErr), zap.String("user_id", user.ID))

			} else {

				emailConfirmed = true

			}

		} else {

			emailConfirmed = true

		}

		return user.ID, user.Role, emailConfirmed, nil

	}

	return "", "", false, err

}

func (s *userServer) linkGoogleToEmailUser(ctx context.Context, googleSub, emailHash string) (userID, role string, emailConfirmed bool, err error) {

	user, err := s.userRepo.GetByEmailHash(ctx, emailHash)

	if err == nil {

		if linkErr := s.userRepo.LinkGoogleAccount(ctx, googleSub, user.ID); linkErr != nil {

			s.log.Warn("Failed to link Google account", zap.Error(linkErr), zap.String("user_id", user.ID))

		}

		return user.ID, user.Role, user.EmailVerified, nil

	}

	return "", "", false, err

}

func (s *userServer) createGoogleUser(ctx context.Context, googleSub, emailHash, emailVal string) (userID, role string, emailConfirmed bool, err error) {

	nickname := extractLocalPart(emailVal)

	nicknameHash := db.NicknameHash(nickname)

	userID, role, emailConfirmed, err = s.userRepo.CreateGoogleUser(ctx, googleSub, emailHash, emailVal, nickname, nicknameHash)

	if err != nil {

		s.log.Error("Failed to create OAuth user", zap.Error(err))

		return "", "", false, status.Error(codes.Internal, "failed to create user")

	}

	if err := s.profileRepo.CreateProfile(ctx, userID); err != nil {

		s.log.Warn("Failed to create profile for OAuth user", zap.Error(err), zap.String("user_id", userID))

	}

	return userID, role, emailConfirmed, nil

}

func (s *userServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.UserProfile, error) {

	user, err := s.userRepo.GetProfileWithPgsodium(ctx, req.UserId)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Database error getting profile", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	profile := &pb.UserProfile{

		UserId: user.ID,

		Email: user.Email,

		FullName: user.FullName,

		Nickname: user.Nickname,

		ProfilePhotoUrl: user.ProfilePhotoURL,

		Role: user.Role,

		Age: user.Age,

		Gender: user.Gender,

		HeightCm: user.HeightCm,

		WeightKg: user.WeightKg,

		FitnessLevel: user.FitnessLevel,

		Goals: user.Goals,

		Nutrition: user.Nutrition,

		SleepHours: user.SleepHours,

		CreatedAt: user.CreatedAt.Format(time.RFC3339),

		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIAccess, "get_profile", "user").
			WithUser(user.ID).
			WithFields("email", "full_name", "nickname", "age", "gender", "height_cm", "weight_kg", "fitness_level", "goals", "nutrition", "sleep_hours"))

	}

	return profile, nil

}

func (s *userServer) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.UserProfile, error) {

	if req.Email == "" {

		return nil, status.Error(codes.InvalidArgument, "email is required")

	}

	user, err := s.userRepo.GetByEmail(ctx, req.Email)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Database error getting user by email", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	profile := &pb.UserProfile{

		UserId: user.ID,

		Email: user.Email,

		FullName: user.FullName,

		Nickname: user.Nickname,

		Role: user.Role,

		CreatedAt: user.CreatedAt.Format(time.RFC3339),

		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}

	return profile, nil

}

func (s *userServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {

	if req.RefreshToken == "" {

		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")

	}

	rt, err := s.refreshTokenRepo.GetValid(ctx, req.RefreshToken)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")

		}

		s.log.Error("Database error checking refresh token", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if rt.ExpiresAt.Before(time.Now()) {

		return nil, status.Error(codes.Unauthenticated, "refresh token expired")

	}

	err = s.refreshTokenRepo.MarkRevoked(ctx, req.RefreshToken)

	if err != nil {

		s.log.Error("Failed to mark refresh token as used", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	user, err := s.userRepo.GetByID(ctx, rt.UserID)

	if err != nil {

		s.log.Error("Failed to get user for refresh", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	accessToken, err := s.tokenProvider.GenerateAccessToken(user.ID, user.Email, user.Role, 15*time.Minute)

	if err != nil {

		s.log.Error("Failed to generate access token", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to generate access token")

	}

	newRefresh := s.tokenProvider.GenerateRefreshToken()

	newRT := &port.RefreshToken{

		UserID: rt.UserID,

		Token: newRefresh,

		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.refreshTokenRepo.Create(ctx, newRT); err != nil {

		s.log.Error("Failed to store new refresh token", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	return &pb.RefreshTokenResponse{

		AccessToken: accessToken,

		TokenType: "Bearer",

		ExpiresIn: 900,

		UserId: user.ID,

		Role: user.Role,

		RefreshToken: newRefresh,
	}, nil

}

func (s *userServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserProfile, error) {

	if err := validator.ValidateProfileUpdate(req); err != nil {

		s.log.Warn("Invalid profile update request", zap.Error(err))

		return nil, fmt.Errorf("validate profile update: %w", err)

	}

	if req.FullName != nil || req.Nickname != nil {

		if err := s.userRepo.UpdateUserDetails(ctx, req.UserId, toString(req.FullName), toString(req.Nickname)); err != nil {

			s.log.Error("Failed to update user details", zap.Error(err), zap.String("user_id", req.UserId))

			return nil, status.Error(codes.Internal, "failed to update user details")

		}

	}

	var userExists bool

	userExists, err := s.profileRepo.UserExists(ctx, req.UserId)

	if err != nil {

		s.log.Error("Failed to check user existence", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if !userExists {

		s.log.Error("User not found during profile update", zap.String("user_id", req.UserId))

		return nil, status.Error(codes.NotFound, errUserNotFound)

	}

	if err := s.profileRepo.UpsertProfile(ctx, req.UserId, &port.ProfileData{

		Age: toInt32(req.Age),

		Gender: toString(req.Gender),

		HeightCm: toInt32(req.HeightCm),

		WeightKg: toFloat64(req.WeightKg),

		FitnessLevel: toString(req.FitnessLevel),

		Nutrition: toString(req.Nutrition),

		SleepHours: toFloat32(req.SleepHours),
	}); err != nil {

		s.log.Error("Failed to update profile", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to update profile")

	}

	if err := s.userRepo.ReplaceUserGoals(ctx, req.UserId, req.Goals); err != nil {

		return nil, err

	}

	if err := s.userRepo.ReplaceUserContraindications(ctx, req.UserId, req.Contraindications); err != nil {

		return nil, err

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIUpdate, "update_profile", "user").
			WithUser(req.UserId).
			WithFields("full_name", "nickname", "age", "gender", "height_cm", "weight_kg", "fitness_level", "goals", "nutrition", "sleep_hours", "contraindications"))

	}

	return s.GetProfile(ctx, &pb.GetProfileRequest{UserId: req.UserId})

}

// ChangePassword changes the user's password after verifying the current one.

func (s *userServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if req.CurrentPassword == "" {

		return nil, status.Error(codes.InvalidArgument, "current_password is required")

	}

	if req.NewPassword == "" {

		return nil, status.Error(codes.InvalidArgument, "new_password is required")

	}

	// Validate new password complexity

	if len(req.NewPassword) < 8 {

		return nil, status.Error(codes.InvalidArgument, "new password must be at least 8 characters")

	}

	// Check password strength (uppercase, lowercase, digit)

	if !containsUpperCase(req.NewPassword) || !containsLowerCase(req.NewPassword) || !containsDigit(req.NewPassword) {

		return nil, status.Error(codes.InvalidArgument, "new password must contain uppercase, lowercase, and digit")

	}

	// Fetch current password hash

	currentHash, err := s.userRepo.GetPasswordHash(ctx, req.UserId)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Failed to fetch password hash", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	// Verify current password

	if !verifyPasswordArgon2id(currentHash, req.CurrentPassword) {

		s.log.Warn("Password change failed: incorrect current password", zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Unauthenticated, "current password is incorrect")

	}

	// Hash new password

	newHash, err := hashPasswordArgon2id(req.NewPassword)

	if err != nil {

		s.log.Error("Failed to hash new password", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to hash new password")

	}

	// Update password

	if err := s.userRepo.UpdatePassword(ctx, req.UserId, string(newHash)); err != nil {

		s.log.Error("Failed to update password", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to update password")

	}

	s.log.Info("Password changed successfully", zap.String("user_id", req.UserId))

	return &pb.ChangePasswordResponse{Message: "Password changed successfully"}, nil

}

// ChangeNickname changes the user's nickname.

func (s *userServer) ChangeNickname(ctx context.Context, req *pb.ChangeNicknameRequest) (*pb.ChangeNicknameResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if req.NewNickname == "" {

		return nil, status.Error(codes.InvalidArgument, "new_nickname is required")

	}

	var exists bool

	var err error

	exists, err = s.userRepo.NicknameExists(ctx, req.NewNickname, req.UserId)

	if err != nil {

		s.log.Error("Failed to check nickname uniqueness", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if exists {

		return nil, status.Error(codes.AlreadyExists, "nickname already taken")

	}

	nicknameHash := db.NicknameHash(req.NewNickname)

	nicknameNonce, err := db.GenerateNonce()

	if err != nil {

		s.log.Error(logFailedToGenerateNonce, zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToGenerateNonce)

	}

	if err := s.userRepo.UpdateNicknameWithPgsodium(ctx, req.UserId, req.NewNickname, nicknameNonce, nicknameHash); err != nil {

		s.log.Error("Failed to update nickname", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to update nickname")

	}

	s.log.Info("Nickname changed", zap.String("user_id", req.UserId), zap.String("new_nickname", req.NewNickname))

	return &pb.ChangeNicknameResponse{Message: "Nickname changed successfully"}, nil

}

// UploadProfilePhoto uploads a new profile photo for the user.

func (s *userServer) UploadProfilePhoto(ctx context.Context, req *pb.UploadProfilePhotoRequest) (*pb.UploadProfilePhotoResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if len(req.PhotoData) == 0 {

		return nil, status.Error(codes.InvalidArgument, "photo_data is required")

	}

	// Validate content type

	if req.ContentType != "image/jpeg" && req.ContentType != "image/png" && req.ContentType != "image/gif" {

		return nil, status.Error(codes.InvalidArgument, "unsupported content type")

	}

	// Generate filename

	filename := req.UserId + "_profile." + strings.TrimPrefix(req.ContentType, "image/")

	// In production, save to storage like S3

	// For now, simulate by updating DB with URL

	photoURL := s.baseURL + "/uploads/profile_photos/" + filename

	if err := s.userRepo.UpdateProfilePhoto(ctx, req.UserId, photoURL); err != nil {

		s.log.Error("Failed to update profile photo URL", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to update profile photo")

	}

	s.log.Info("Profile photo uploaded", zap.String("user_id", req.UserId), zap.String("photo_url", photoURL))

	return &pb.UploadProfilePhotoResponse{PhotoUrl: photoURL}, nil

}

// RemoveProfilePhoto removes the user's profile photo.

func (s *userServer) RemoveProfilePhoto(ctx context.Context, req *pb.RemoveProfilePhotoRequest) (*pb.RemoveProfilePhotoResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	// Update DB to remove photo URL

	if err := s.userRepo.RemoveProfilePhoto(ctx, req.UserId); err != nil {

		s.log.Error("Failed to remove profile photo", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to remove profile photo")

	}

	s.log.Info("Profile photo removed", zap.String("user_id", req.UserId))

	return &pb.RemoveProfilePhotoResponse{Message: "Profile photo removed successfully"}, nil

}

// ListDevices lists the user's connected devices.

func (s *userServer) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if s.userSvc != nil {

		devices, err := s.userSvc.ListDevices(ctx, req.UserId)

		if err != nil {

			s.log.Error("Failed to list devices", zap.Error(err))

			return nil, status.Error(codes.Internal, errFailedToListDevices)

		}

		var pbDevices []*pb.Device

		for _, d := range devices {

			pbDevices = append(pbDevices, &pb.Device{

				DeviceId: d.ID,

				DeviceType: d.DeviceType,

				DeviceName: d.DeviceName,

				IsConnected: d.IsConnected,

				LastSync: d.LastSync.Format(time.RFC3339),
			})

		}

		return &pb.ListDevicesResponse{Devices: pbDevices}, nil

	}

	devices, err := s.userSvc.ListDevices(ctx, req.UserId)

	if err != nil {

		s.log.Error("Failed to list devices", zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToListDevices)

	}

	var pbDevices []*pb.Device

	for _, d := range devices {

		pbDevices = append(pbDevices, &pb.Device{

			DeviceId: d.ID,

			DeviceType: d.DeviceType,

			DeviceName: d.DeviceName,

			IsConnected: d.IsConnected,

			LastSync: d.LastSync.Format(time.RFC3339),
		})

	}

	return &pb.ListDevicesResponse{Devices: pbDevices}, nil

}

// AddDevice adds a new device for the user.

func (s *userServer) AddDevice(ctx context.Context, req *pb.AddDeviceRequest) (*pb.AddDeviceResponse, error) {

	if req.UserId == "" || req.DeviceType == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and device_type are required")

	}

	device := &entity.Device{

		ID: uuid.New().String(),

		UserID: req.UserId,

		DeviceType: req.DeviceType,

		DeviceName: req.DeviceName,

		Token: uuid.New().String(),
	}

	_, err := s.userSvc.AddDevice(ctx, device)

	if err != nil {

		s.log.Error("Failed to add device", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to add device")

	}

	s.log.Info("Device added", zap.String("user_id", req.UserId), zap.String("device_id", device.ID))

	return &pb.AddDeviceResponse{

		Device: &pb.Device{

			DeviceId: device.ID,

			DeviceType: device.DeviceType,

			DeviceName: device.DeviceName,

			IsConnected: true,

			LastSync: time.Now().Format(time.RFC3339),
		},
	}, nil

}

// RemoveDevice removes a device for the user.

func (s *userServer) RemoveDevice(ctx context.Context, req *pb.RemoveDeviceRequest) (*pb.RemoveDeviceResponse, error) {

	if req.UserId == "" || req.DeviceId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and device_id are required")

	}

	if err := s.userSvc.RemoveDevice(ctx, req.UserId, req.DeviceId); err != nil {

		s.log.Error("Failed to remove device", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to remove device")

	}

	s.log.Info("Device removed", zap.String("user_id", req.UserId), zap.String("device_id", req.DeviceId))

	return &pb.RemoveDeviceResponse{Message: "Device removed successfully"}, nil

}

// SyncDeviceData syncs data from the device.

func (s *userServer) SyncDeviceData(ctx context.Context, req *pb.SyncDeviceDataRequest) (*pb.SyncDeviceDataResponse, error) {

	if req.UserId == "" || req.DeviceId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and device_id are required")

	}

	return nil, status.Error(codes.Unimplemented, "SyncDeviceData is not yet implemented")

}

// GetTrainingStats retrieves training statistics for the user.

func (s *userServer) GetTrainingStats(ctx context.Context, req *pb.GetTrainingStatsRequest) (*pb.GetTrainingStatsResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	return nil, status.Error(codes.Unimplemented, "GetTrainingStats is not yet implemented")

}

// GetAchievements retrieves all achievements with user's earned status from database.

func (s *userServer) GetAchievements(ctx context.Context, req *pb.GetAchievementsRequest) (*pb.GetAchievementsResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	achievements, err := s.achievementExRepo.ListWithEarnedStatus(ctx, req.UserId)

	if err != nil {

		s.log.Error("Failed to query achievements", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to query achievements")

	}

	var pbAchievements []*pb.Achievement

	for _, a := range achievements {

		earnedDate := ""

		if a.EarnedAt != nil {

			earnedDate = a.EarnedAt.Format(time.RFC3339)

		}

		pbAchievements = append(pbAchievements, &pb.Achievement{

			AchievementId: a.ID,

			Title: a.Name,

			Description: a.Description,

			EarnedDate: earnedDate,

			IconUrl: a.IconURL,
		})

	}

	if pbAchievements == nil {

		pbAchievements = []*pb.Achievement{}

	}

	return &pb.GetAchievementsResponse{Achievements: pbAchievements}, nil

}

// Helper functions for password validation

func containsUpperCase(s string) bool {

	for _, r := range s {

		if r >= 'A' && r <= 'Z' {

			return true

		}

	}

	return false

}

func containsLowerCase(s string) bool {

	for _, r := range s {

		if r >= 'a' && r <= 'z' {

			return true

		}

	}

	return false

}

func containsDigit(s string) bool {

	for _, r := range s {

		if r >= '0' && r <= '9' {

			return true

		}

	}

	return false

}

func extractLocalPart(email string) string {

	parts := strings.Split(email, "@")

	if len(parts) > 0 {

		return parts[0]

	}

	return email

}

func safeInt32(v int) int32 {

	if v > math.MaxInt32 {

		return math.MaxInt32

	}

	if v < math.MinInt32 {

		return math.MinInt32

	}

	return int32(v)

}

// generateVerificationToken generates a random 32-byte hex token for email verification.

func generateVerificationToken() string {

	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {

		panic("failed to generate verification token: " + err.Error())

	}

	return hex.EncodeToString(b)

}

func (s *userServer) RegisterWithInvite(ctx context.Context, req *pb.RegisterWithInviteRequest) (*pb.RegisterResponse, error) {

	s.log.Info("Register with invite code", zap.String("email", req.GetEmail()))

	// Валидация invite-кода

	isValid, role, specialty, errMsg, err := s.inviteCodeRepo.ValidateInviteCodeUse(ctx, req.GetInviteCode())

	if err != nil {

		s.log.Error("Failed to validate invite code", zap.Error(err))

		return nil, status.Error(codes.Internal, errInternalError)

	}

	if !isValid {

		return nil, status.Errorf(codes.InvalidArgument, "invite code error: %s", errMsg)

	}

	_ = specialty

	// Определяем роль: приоритет у invite_code role

	finalRole := role

	// Хешируем пароль

	hashedPassword, err := hashPasswordArgon2id(req.GetPassword())

	if err != nil {

		s.log.Error("Failed to hash password", zap.Error(err))

		return nil, status.Error(codes.Internal, errInternalError)

	}

	// Создаём пользователя

	userID := uuid.New().String()

	emailVal := sanitize.String(req.GetEmail())

	fullName := sanitize.String(req.GetFullName())

	hashedPasswordStr := string(hashedPassword)

	err = s.userRepo.CreateWithInvite(ctx, userID, emailVal, db.EmailHash(emailVal), hashedPasswordStr, fullName, finalRole)

	if err != nil {

		if apperrors.IsConflict(err) {

			return nil, status.Error(codes.AlreadyExists, "email already exists")

		}

		s.log.Error("Failed to create user", zap.Error(err))

		return nil, status.Error(codes.Internal, errInternalError)

	}

	// Генерируем JWT (токен возвращается при login, не при регистрации)

	_, err = s.tokenProvider.GenerateAccessToken(userID, req.GetEmail(), finalRole, 15*time.Minute)

	if err != nil {

		s.log.Error(logFailedToGenerateJWT, zap.Error(err))

		return nil, status.Error(codes.Internal, errInternalError)

	}

	s.log.Info("User registered via invite code",

		zap.String("user_id", userID),

		zap.String("email", req.GetEmail()),

		zap.String("role", finalRole),
	)

	return &pb.RegisterResponse{

		UserId: userID,

		Message: "Регистрация успешна",
	}, nil

}

func (s *userServer) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {

	if req.GetCode() == "" {

		return nil, status.Error(codes.InvalidArgument, "code is required")

	}

	isValid, role, specialty, errMsg, err := s.inviteCodeRepo.ValidateInviteCodeUse(ctx, req.GetCode())

	if err != nil {

		s.log.Error("Failed to validate invite code", zap.Error(err))

		return nil, status.Error(codes.Internal, errInternalError)

	}

	return &pb.ValidateInviteCodeResponse{

		IsValid: isValid,

		Role: role,

		Specialty: specialty,

		ErrorMessage: errMsg,
	}, nil

}

func (s *userServer) SetupTOTP(ctx context.Context, req *pb.SetupTOTPRequest) (*pb.SetupTOTPResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	email, totpEnabled, err := s.userRepo.GetEmailByID(ctx, req.UserId)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Database error getting email for TOTP setup", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if totpEnabled {

		return nil, status.Error(codes.AlreadyExists, "2FA already enabled")

	}

	setup, err := s.totpService.GenerateTOTPSecret(email)

	if err != nil {

		s.log.Error("Failed to generate TOTP secret", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to generate TOTP secret")

	}

	s.log.Info("TOTP setup generated", zap.String("user_id", req.UserId))

	return &pb.SetupTOTPResponse{

		QrCodeUrl: setup.QRCodeURL,

		Secret: setup.Secret,

		BackupCodes: setup.BackupCodes,
	}, nil

}

func (s *userServer) ConfirmTOTP(ctx context.Context, req *pb.ConfirmTOTPRequest) (*pb.ConfirmTOTPResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if req.TempSecret == "" || req.Passcode == "" {

		return nil, status.Error(codes.InvalidArgument, "temp_secret and passcode are required")

	}

	if len(req.BackupCodes) != totp.BackupCodesCount {

		return nil, status.Error(codes.InvalidArgument, "exactly 10 backup codes are required")

	}

	valid, err := s.totpService.ValidateTOTPCode(req.Passcode, req.TempSecret)

	if err != nil {

		s.log.Warn("TOTP code validation error", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errFailedToValidateTOTP)

	}

	if !valid {

		return &pb.ConfirmTOTPResponse{Success: false, Message: "Invalid TOTP code"}, nil

	}

	encryptedSecret, err := s.totpService.EncryptSecret(req.TempSecret)

	if err != nil {

		s.log.Error("Failed to encrypt TOTP secret", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to encrypt secret")

	}

	hashedCodes := totp.HashBackupCodes(req.BackupCodes)

	if err := s.userRepo.EnableTOTP(ctx, req.UserId, encryptedSecret, hashedCodes, intToInt32(len(req.BackupCodes))); err != nil {

		s.log.Error("Failed to enable TOTP", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to enable TOTP")

	}

	s.log.Info("TOTP enabled successfully", zap.String("user_id", req.UserId))

	return &pb.ConfirmTOTPResponse{Success: true, Message: "TOTP enabled successfully"}, nil

}

func (s *userServer) VerifyTOTP(ctx context.Context, req *pb.VerifyTOTPRequest) (*pb.VerifyTOTPResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if req.Passcode == "" {

		return nil, status.Error(codes.InvalidArgument, "passcode is required")

	}

	encryptedSecret, totpEnabled, backupCodesHash, backupCodesRemaining, err := s.userRepo.GetTOTPState(ctx, req.UserId)

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if !totpEnabled {

		return &pb.VerifyTOTPResponse{Valid: false}, nil

	}

	if req.IsBackupCode {

		idx, backupErr := totp.ValidateBackupCode(req.Passcode, backupCodesHash)

		if backupErr != nil {

			s.log.Warn("Invalid backup code", zap.Error(backupErr), zap.String("user_id", req.UserId))

			return nil, status.Error(codes.Unauthenticated, "invalid backup code")

		}

		remaining := backupCodesRemaining

		err = s.userRepo.RemoveBackupCode(ctx, req.UserId, backupCodesHash[idx])

		if err != nil {

			s.log.Warn("Failed to remove used backup code", zap.Error(err))

		} else if remaining > 0 {

			remaining--

		}

		return &pb.VerifyTOTPResponse{Valid: true, BackupCodesRemaining: remaining}, nil

	}

	secret, err := s.totpService.DecryptSecret(encryptedSecret)

	if err != nil {

		s.log.Error("Failed to decrypt TOTP secret", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to decrypt secret")

	}

	valid, err := s.totpService.ValidateTOTPCode(req.Passcode, secret)

	if err != nil {

		s.log.Warn("TOTP validation error", zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToValidateTOTP)

	}

	s.log.Info("TOTP verified", zap.String("user_id", req.UserId), zap.Bool("valid", valid))

	return &pb.VerifyTOTPResponse{Valid: valid, BackupCodesRemaining: backupCodesRemaining}, nil

}

func (s *userServer) DisableTOTP(ctx context.Context, req *pb.DisableTOTPRequest) (*pb.DisableTOTPResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if req.Passcode == "" {

		return nil, status.Error(codes.InvalidArgument, "passcode is required")

	}

	encryptedSecret, totpEnabled, _, _, err := s.userRepo.GetTOTPState(ctx, req.UserId)

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {

			return nil, status.Error(codes.NotFound, "TOTP not enabled for user")

		}

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if !totpEnabled {

		return nil, status.Error(codes.NotFound, "TOTP not enabled for user")

	}

	secret, err := s.totpService.DecryptSecret(encryptedSecret)

	if err != nil {

		s.log.Error("Failed to decrypt TOTP secret for disable", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to decrypt secret")

	}

	valid, err := s.totpService.ValidateTOTPCode(req.Passcode, secret)

	if err != nil {

		s.log.Warn("TOTP validation error during disable", zap.Error(err))

		return nil, status.Error(codes.Internal, errFailedToValidateTOTP)

	}

	if !valid {

		return &pb.DisableTOTPResponse{Success: false, Message: "Invalid TOTP code"}, nil

	}

	if err := s.userRepo.DisableTOTP(ctx, req.UserId); err != nil {

		s.log.Error("Failed to disable TOTP", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to disable TOTP")

	}

	s.log.Info("TOTP disabled", zap.String("user_id", req.UserId))

	return &pb.DisableTOTPResponse{Success: true, Message: "TOTP disabled successfully"}, nil

}

func (s *userServer) ListHealthConditions(ctx context.Context, req *pb.ListHealthConditionsRequest) (*pb.ListHealthConditionsResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	conditions, err := s.userHealthRepo.List(ctx, req.UserId)

	if err != nil {

		s.log.Error("Failed to query health conditions", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	var pbConditions []*pb.HealthCondition

	for _, c := range conditions {

		diagnosedAt := ""

		if c.DiagnosedAt != nil {

			diagnosedAt = c.DiagnosedAt.Format(dateFormat)

		}

		notes := ""

		if c.Notes != nil {

			notes = *c.Notes

		}

		pbConditions = append(pbConditions, &pb.HealthCondition{

			Id: c.ID, UserId: c.UserID, ConditionType: c.ConditionType, ConditionName: c.ConditionName,

			Severity: c.Severity, DiagnosedAt: diagnosedAt, IsActive: c.IsActive, Notes: notes,

			CreatedAt: c.CreatedAt.Format(time.RFC3339), UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		})

	}

	if pbConditions == nil {

		pbConditions = []*pb.HealthCondition{}

	}

	return &pb.ListHealthConditionsResponse{Conditions: pbConditions, Total: safeInt32(len(pbConditions))}, nil

}

func (s *userServer) UpsertHealthCondition(ctx context.Context, req *pb.UpsertHealthConditionRequest) (*pb.HealthCondition, error) {

	if req.UserId == "" || req.ConditionName == "" || req.ConditionType == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id, condition_type and condition_name are required")

	}

	condition := &port.UserHealthCondition{

		UserID: req.UserId,

		ConditionType: req.ConditionType,

		ConditionName: req.ConditionName,

		Severity: req.Severity,

		IsActive: req.IsActive,
	}

	if req.DiagnosedAt != "" {

		t, parseErr := time.Parse(dateFormat, req.DiagnosedAt)

		if parseErr != nil {

			return nil, status.Error(codes.InvalidArgument, "invalid diagnosed_at format, use YYYY-MM-DD")

		}

		condition.DiagnosedAt = &t

	}

	if req.Notes != "" {

		notes := req.Notes

		condition.Notes = &notes

	}

	result, err := s.userHealthRepo.Upsert(ctx, condition)

	if err != nil {

		s.log.Error("Failed to upsert health condition", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIUpdate, "upsert_health_condition", "health_condition").
			WithUser(req.UserId).
			WithFields("condition_type", "condition_name", "severity", "is_active"))

	}

	return &pb.HealthCondition{

		Id: result.ID, UserId: result.UserID, ConditionType: result.ConditionType, ConditionName: result.ConditionName,

		Severity: result.Severity, DiagnosedAt: req.DiagnosedAt, IsActive: result.IsActive, Notes: req.Notes,
	}, nil

}

func (s *userServer) DeleteHealthCondition(ctx context.Context, req *pb.DeleteHealthConditionRequest) (*pb.DeleteHealthConditionResponse, error) {

	if req.UserId == "" || req.ConditionId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and condition_id are required")

	}

	if err := s.userRepo.DeleteHealthCondition(ctx, req.ConditionId, req.UserId); err != nil {

		return nil, err

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIDelete, "delete_health_condition", "health_condition").
			WithUser(req.UserId).
			WithFields("condition_id"))

	}

	return &pb.DeleteHealthConditionResponse{Success: true, Message: "Health condition deleted"}, nil

}

func parseTimeInput(value string) (*time.Time, error) {

	if value == "" {

		return nil, nil

	}

	t, err := time.Parse(time.RFC3339, value)

	if err != nil {

		t, err = time.Parse(dateFormat, value)

		if err != nil {

			return nil, status.Error(codes.InvalidArgument, "invalid date format")

		}

	}

	return &t, nil

}

func (s *userServer) ListBodyComposition(ctx context.Context, req *pb.ListBodyCompositionRequest) (*pb.ListBodyCompositionResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	limit := req.Limit

	if limit <= 0 {

		limit = 100

	}

	limitClamped := limit

	if limitClamped > 10000 {

		limitClamped = 10000

	}

	from, err := parseTimeInput(req.From)

	if err != nil {

		return nil, err

	}

	to, err := parseTimeInput(req.To)

	if err != nil {

		return nil, err

	}

	records, err := s.userBodyCompRepo.List(ctx, req.UserId, from, to, int(limitClamped))

	if err != nil {

		s.log.Error("Failed to query body composition", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	pbRecords := make([]*pb.BodyCompositionRecord, 0)

	for _, r := range records {

		pbRecords = append(pbRecords, &pb.BodyCompositionRecord{

			Id: r.ID, UserId: r.UserID, RecordedAt: r.RecordedAt.Format(time.RFC3339), WeightKg: r.WeightKG,

			HeightCm: int32(r.HeightCM), Bmi: r.BMI,

			BodyFatPercentage: toFloat64(r.BodyFatPercentage), MuscleMassPercentage: toFloat64(r.MuscleMassPercentage),

			BoneMassPercentage: toFloat64(r.BoneMassPercentage), WaterPercentage: toFloat64(r.WaterPercentage),

			VisceralFatRating: int32(toFloat64(r.VisceralFatRating)), MetabolicAge: int32(toFloat64(r.MetabolicAge)),

			Source: r.Source,
		})

	}

	return &pb.ListBodyCompositionResponse{Records: pbRecords, Total: safeInt32(len(pbRecords))}, nil

}

func (s *userServer) CreateBodyComposition(ctx context.Context, req *pb.CreateBodyCompositionRequest) (*pb.BodyCompositionRecord, error) {

	if req.UserId == "" || req.WeightKg <= 0 {

		return nil, status.Error(codes.InvalidArgument, "user_id and weight_kg are required")

	}

	bc := &port.UserBodyComposition{

		UserID: req.UserId,

		WeightKG: req.WeightKg,

		HeightCM: float64(req.HeightCm),

		BMI: req.Bmi,

		BodyFatPercentage: float64Ptr(req.BodyFatPercentage),

		MuscleMassPercentage: float64Ptr(req.MuscleMassPercentage),

		BoneMassPercentage: float64Ptr(req.BoneMassPercentage),

		WaterPercentage: float64Ptr(req.WaterPercentage),

		VisceralFatRating: float64Ptr(float64(req.VisceralFatRating)),

		MetabolicAge: float64Ptr(float64(req.MetabolicAge)),

		Source: req.Source,
	}

	if req.RecordedAt != "" {

		t, parseErr := time.Parse(time.RFC3339, req.RecordedAt)

		if parseErr != nil {

			t, parseErr = time.Parse(dateFormat, req.RecordedAt)

			if parseErr != nil {

				return nil, status.Error(codes.InvalidArgument, "invalid recorded_at format")

			}

		}

		bc.RecordedAt = t

	} else {

		bc.RecordedAt = time.Now()

	}

	result, err := s.userBodyCompRepo.Create(ctx, bc)

	if err != nil {

		s.log.Error("Failed to create body composition record", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIUpdate, "create_body_composition", "body_composition").
			WithUser(req.UserId).
			WithFields("weight_kg", "height_cm", "bmi", "body_fat_percentage", "muscle_mass_percentage", "bone_mass_percentage", "water_percentage", "visceral_fat_rating", "metabolic_age"))

	}

	return &pb.BodyCompositionRecord{

		Id: result.ID, UserId: result.UserID, RecordedAt: result.RecordedAt.Format(time.RFC3339), WeightKg: result.WeightKG,

		HeightCm: int32(result.HeightCM), Bmi: result.BMI,

		BodyFatPercentage: toFloat64(result.BodyFatPercentage), MuscleMassPercentage: toFloat64(result.MuscleMassPercentage),

		BoneMassPercentage: toFloat64(result.BoneMassPercentage), WaterPercentage: toFloat64(result.WaterPercentage),

		VisceralFatRating: int32(toFloat64(result.VisceralFatRating)), MetabolicAge: int32(toFloat64(result.MetabolicAge)),

		Source: result.Source,
	}, nil

}

func (s *userServer) ListMenstrualCycles(ctx context.Context, req *pb.ListMenstrualCyclesRequest) (*pb.ListMenstrualCyclesResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	cycles, err := s.userMenstrualRepo.ListCycles(ctx, req.UserId)

	if err != nil {

		s.log.Error("Failed to query menstrual cycles", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	pbCycles := make([]*pb.MenstrualCycle, 0)

	for _, c := range cycles {

		symptoms, err := s.userMenstrualRepo.ListSymptoms(ctx, c.ID)

		if err != nil {

			s.log.Error("Failed to query menstrual symptoms", zap.Error(err), zap.String("cycle_id", c.ID))

			return nil, status.Error(codes.Internal, errDatabaseError)

		}

		moods, err := s.userMenstrualRepo.ListMoods(ctx, c.ID)

		if err != nil {

			s.log.Error("Failed to query menstrual moods", zap.Error(err), zap.String("cycle_id", c.ID))

			return nil, status.Error(codes.Internal, errDatabaseError)

		}

		pbCycles = append(pbCycles, &pb.MenstrualCycle{

			Id: c.ID, UserId: c.UserID, CycleStartDate: c.CycleStartDate,

			CycleEndDate: c.CycleEndDate, FlowIntensity: c.FlowIntensity,

			Notes: c.Notes, Symptoms: symptoms, Moods: moods,

			CreatedAt: c.CreatedAt.Format(time.RFC3339), UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		})

	}

	return &pb.ListMenstrualCyclesResponse{Cycles: pbCycles, Total: safeInt32(len(pbCycles))}, nil

}

func (s *userServer) CreateMenstrualCycle(ctx context.Context, req *pb.CreateMenstrualCycleRequest) (*pb.MenstrualCycle, error) {

	if req.UserId == "" || req.CycleStartDate == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and cycle_start_date are required")

	}

	cycle := &port.UserMenstrualCycle{

		UserID: req.UserId,

		CycleStartDate: req.CycleStartDate,

		CycleEndDate: req.CycleEndDate,

		FlowIntensity: req.FlowIntensity,

		Notes: req.Notes,

		Symptoms: req.Symptoms,

		Moods: req.Moods,
	}

	result, err := s.userMenstrualRepo.CreateCycleWithDetails(ctx, cycle)

	if err != nil {

		s.log.Error("Failed to create menstrual cycle", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIUpdate, "create_menstrual_cycle", "menstrual_cycle").
			WithUser(req.UserId).
			WithFields("cycle_start_date", "cycle_end_date", "flow_intensity", "notes", "symptoms", "moods"))

	}

	return &pb.MenstrualCycle{

		Id: result.ID, UserId: result.UserID, CycleStartDate: result.CycleStartDate,

		CycleEndDate: result.CycleEndDate, FlowIntensity: result.FlowIntensity,

		Notes: result.Notes, Symptoms: result.Symptoms, Moods: result.Moods,
	}, nil

}

func (s *userServer) UpdateMenstrualCycle(ctx context.Context, req *pb.UpdateMenstrualCycleRequest) (*pb.MenstrualCycle, error) {

	if req.UserId == "" || req.CycleId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and cycle_id are required")

	}

	cycle := &port.UserMenstrualCycle{

		ID: req.CycleId,

		UserID: req.UserId,

		CycleStartDate: req.CycleStartDate,

		CycleEndDate: req.CycleEndDate,

		FlowIntensity: req.FlowIntensity,

		Notes: req.Notes,

		Symptoms: req.Symptoms,

		Moods: req.Moods,
	}

	result, err := s.userMenstrualRepo.UpdateCycleWithDetails(ctx, cycle)

	if err != nil {

		s.log.Error("Failed to update menstrual cycle", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIUpdate, "update_menstrual_cycle", "menstrual_cycle").
			WithUser(req.UserId).
			WithFields("cycle_id", "cycle_start_date", "cycle_end_date", "flow_intensity", "notes", "symptoms", "moods"))

	}

	return &pb.MenstrualCycle{

		Id: result.ID, UserId: result.UserID, CycleStartDate: result.CycleStartDate,

		CycleEndDate: result.CycleEndDate, FlowIntensity: result.FlowIntensity,

		Notes: result.Notes, Symptoms: result.Symptoms, Moods: result.Moods,
	}, nil

}

func (s *userServer) DeleteMenstrualCycle(ctx context.Context, req *pb.DeleteMenstrualCycleRequest) (*pb.DeleteMenstrualCycleResponse, error) {

	if req.UserId == "" || req.CycleId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id and cycle_id are required")

	}

	if err := s.userRepo.DeleteMenstrualCycle(ctx, req.CycleId, req.UserId); err != nil {

		return nil, err

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIDelete, "delete_menstrual_cycle", "menstrual_cycle").
			WithUser(req.UserId).
			WithFields("cycle_id"))

	}

	return &pb.DeleteMenstrualCycleResponse{Success: true, Message: "Menstrual cycle deleted"}, nil

}

func (s *userServer) GetUserClaims(ctx context.Context, req *pb.GetUserClaimsRequest) (*pb.GetUserClaimsResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	email, role, totpEnabled, backupCodesRemaining, err := s.userRepo.GetClaimsByID(ctx, req.UserId)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Failed to get user claims", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	return &pb.GetUserClaimsResponse{

		Email: email,

		Role: role,

		TotpEnabled: totpEnabled,

		TotpBackupCodesRemaining: backupCodesRemaining,
	}, nil

}

func (s *userServer) DeleteProfile(ctx context.Context, req *pb.DeleteProfileRequest) (*pb.DeleteProfileResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	if req.Password == "" {

		return nil, status.Error(codes.InvalidArgument, "password is required")

	}

	var passwordHash string

	var err error

	passwordHash, err = s.userRepo.GetPasswordHash(ctx, req.UserId)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Failed to load user for deletion", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	if !verifyPasswordArgon2id(passwordHash, req.Password) {

		return nil, status.Error(codes.Unauthenticated, "invalid password")

	}

	if err := s.userRepo.DeleteProfileData(ctx, req.UserId); err != nil {

		s.log.Error("Failed to delete profile", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to delete profile")

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIDelete, "delete_profile", "user").
			WithUser(req.UserId).
			WithFields("email", "full_name", "profile", "health_conditions", "body_composition", "menstrual_cycles"))

	}

	s.log.Info("Profile deleted (GDPR)", zap.String("user_id", req.UserId))

	return &pb.DeleteProfileResponse{Status: "deleted", Message: "Profile deleted successfully"}, nil

}

func (s *userServer) ExportUserData(ctx context.Context, req *pb.ExportUserDataRequest) (*pb.ExportUserDataResponse, error) {

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, errUserIDRequired)

	}

	user, err := s.userRepo.GetProfileWithPgsodium(ctx, req.UserId)

	if err != nil {

		if apperrors.IsNotFound(err) {

			return nil, status.Error(codes.NotFound, errUserNotFound)

		}

		s.log.Error("Database error exporting user data", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	conditions, _ := s.userHealthRepo.List(ctx, req.UserId)

	pbConditions := make([]*pb.HealthCondition, 0, len(conditions))

	for _, c := range conditions {

		pbConditions = append(pbConditions, &pb.HealthCondition{

			Id: c.ID, UserId: c.UserID, ConditionType: c.ConditionType, ConditionName: c.ConditionName,

			Severity: c.Severity, DiagnosedAt: "", IsActive: c.IsActive, Notes: toString(c.Notes),
		})

	}

	bodyRecords, _ := s.userBodyCompRepo.List(ctx, req.UserId, nil, nil, 1000)

	pbBodyRecords := make([]*pb.BodyCompositionRecord, 0, len(bodyRecords))

	for _, r := range bodyRecords {

		pbBodyRecords = append(pbBodyRecords, &pb.BodyCompositionRecord{

			Id: r.ID, UserId: r.UserID, RecordedAt: r.RecordedAt.Format(time.RFC3339), WeightKg: r.WeightKG,

			HeightCm: int32(r.HeightCM), Bmi: r.BMI,

			BodyFatPercentage: toFloat64(r.BodyFatPercentage), MuscleMassPercentage: toFloat64(r.MuscleMassPercentage),

			BoneMassPercentage: toFloat64(r.BoneMassPercentage), WaterPercentage: toFloat64(r.WaterPercentage),

			VisceralFatRating: int32(toFloat64(r.VisceralFatRating)), MetabolicAge: int32(toFloat64(r.MetabolicAge)),

			Source: r.Source,
		})

	}

	cycles, _ := s.userMenstrualRepo.ListCycles(ctx, req.UserId)

	pbCycles := make([]*pb.MenstrualCycle, 0, len(cycles))

	for _, c := range cycles {

		symptoms, _ := s.userMenstrualRepo.ListSymptoms(ctx, c.ID)

		moods, _ := s.userMenstrualRepo.ListMoods(ctx, c.ID)

		pbCycles = append(pbCycles, &pb.MenstrualCycle{

			Id: c.ID, UserId: c.UserID, CycleStartDate: c.CycleStartDate,

			CycleEndDate: c.CycleEndDate, FlowIntensity: c.FlowIntensity,

			Notes: c.Notes, Symptoms: symptoms, Moods: moods,

			CreatedAt: c.CreatedAt.Format(time.RFC3339), UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		})

	}

	devices, _ := s.deviceRepo.List(ctx, req.UserId)

	pbDevices := make([]*pb.Device, 0, len(devices))

	for _, d := range devices {

		pbDevices = append(pbDevices, &pb.Device{

			DeviceId: d.ID, DeviceType: d.DeviceType, DeviceName: d.DeviceName,
		})

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypePIIExport, "export_user_data", "user").
			WithUser(req.UserId).
			WithFields("email", "full_name", "profile", "health_conditions", "body_composition", "menstrual_cycles", "devices"))

	}

	return &pb.ExportUserDataResponse{

		UserId: user.ID,

		Email: user.Email,

		FullName: user.FullName,

		Nickname: user.Nickname,

		ProfilePhotoUrl: user.ProfilePhotoURL,

		Role: user.Role,

		Age: user.Age,

		Gender: user.Gender,

		HeightCm: user.HeightCm,

		WeightKg: user.WeightKg,

		FitnessLevel: user.FitnessLevel,

		Goals: user.Goals,

		Contraindications: nil,

		Nutrition: user.Nutrition,

		SleepHours: user.SleepHours,

		CreatedAt: user.CreatedAt.Format(time.RFC3339),

		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),

		HealthConditions: pbConditions,

		BodyComposition: pbBodyRecords,

		MenstrualCycles: pbCycles,

		Devices: pbDevices,

		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil

}

func (s *userServer) requireAdminRole(ctx context.Context, requesterID string) error {

	if requesterID == "" {

		return status.Error(codes.Unauthenticated, "requester_user_id is required")

	}

	var role string

	var err error

	role, err = s.userRepo.GetUserRole(ctx, requesterID)

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {

			return status.Error(codes.NotFound, "requester not found")

		}

		s.log.Error("Failed to verify admin role", zap.Error(err), zap.String("requester_id", requesterID))

		return status.Error(codes.Internal, errDatabaseError)

	}

	if role != "admin" {

		return status.Error(codes.PermissionDenied, "admin role required")

	}

	return nil

}

func (s *userServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {

	if err := s.requireAdminRole(ctx, req.RequesterUserId); err != nil {

		return nil, err

	}

	if req.PageSize <= 0 {

		return nil, status.Error(codes.InvalidArgument, "page_size must be greater than 0")

	}

	if req.Page < 0 {

		return nil, status.Error(codes.InvalidArgument, "page must be non-negative")

	}

	users, total, err := s.userRepo.ListByRole(ctx, req.Role, int(req.Page), int(req.PageSize))

	if err != nil {

		s.log.Error("Failed to list users", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	var pbUsers []*pb.UserProfile

	for _, user := range users {

		pbUsers = append(pbUsers, &pb.UserProfile{

			UserId: user.ID,

			Email: user.Email,

			FullName: user.FullName,

			Nickname: user.Nickname,

			ProfilePhotoUrl: user.ProfilePhotoURL,

			Role: user.Role,

			Age: user.Age,

			Gender: user.Gender,

			HeightCm: user.HeightCm,

			WeightKg: user.WeightKg,

			FitnessLevel: user.FitnessLevel,

			Goals: user.Goals,

			Nutrition: user.Nutrition,

			SleepHours: user.SleepHours,

			CreatedAt: user.CreatedAt.Format(time.RFC3339),

			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		})

	}

	return &pb.ListUsersResponse{

		Users: pbUsers,

		Total: intToInt32(total),
	}, nil

}

func (s *userServer) AdminListInvites(ctx context.Context, req *pb.AdminListInvitesRequest) (*pb.AdminListInvitesResponse, error) {

	if err := s.requireAdminRole(ctx, ""); err != nil {

		return nil, err

	}

	if req.PageSize <= 0 {

		req.PageSize = 20

	}

	if req.Page < 0 {

		req.Page = 0

	}

	inviteCodes, _, err := s.inviteCodeRepo.List(ctx, int(req.Page), int(req.PageSize))

	if err != nil {

		s.log.Error("Failed to list invites", zap.Error(err))

		return nil, status.Error(codes.Internal, errDatabaseError)

	}

	var invites []*pb.InviteInfo

	baseURL := s.baseURL

	if baseURL == "" {

		baseURL = "https://fittpulse.ru"

	}

	for _, inv := range inviteCodes {

		specialty := ""

		if inv.Specialty != nil {

			specialty = *inv.Specialty

		}

		invites = append(invites, &pb.InviteInfo{

			Code: inv.Code,

			Role: inv.Role,

			Specialty: specialty,

			MaxUses: intToInt32(inv.MaxUses),

			UsedCount: intToInt32(inv.UsedCount),

			IsActive: inv.IsActive,

			CreatedAt: inv.CreatedAt.Format(time.RFC3339),

			InviteUrl: fmt.Sprintf("%s/register?invite=%s", baseURL, inv.Code),
		})

	}

	return &pb.AdminListInvitesResponse{

		Invites: invites,

		Total: intToInt32(len(inviteCodes)),
	}, nil

}

func (s *userServer) AdminCreateInvite(ctx context.Context, req *pb.AdminCreateInviteRequest) (*pb.AdminCreateInviteResponse, error) {

	if err := s.requireAdminRole(ctx, ""); err != nil {

		return nil, err

	}

	role := req.GetRole()

	if role == "" {

		role = "client"

	}

	if role != "client" && role != "admin" {

		return nil, status.Error(codes.InvalidArgument, "role must be 'client' or 'admin'")

	}

	maxUses := req.GetMaxUses()

	if maxUses <= 0 {

		maxUses = 1

	}

	if maxUses > 100 {

		return nil, status.Error(codes.InvalidArgument, "max_uses must be between 1 and 100")

	}

	code := "INV-" + generateInviteCode()

	var specialty *string

	if req.GetSpecialty() != "" {

		s := req.GetSpecialty()

		specialty = &s

	}

	invite := &port.InviteCode{

		Code: code,

		Role: role,

		Specialty: specialty,

		MaxUses: int(maxUses),

		UsedCount: 0,

		IsActive: true,

		CreatedBy: "",

		CreatedAt: time.Now(),
	}

	if err := s.inviteCodeRepo.Create(ctx, invite); err != nil {

		s.log.Error("Failed to create invite", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to create invite")

	}

	baseURL := s.baseURL

	if baseURL == "" {

		baseURL = "https://fittpulse.ru"

	}

	inviteURL := fmt.Sprintf("%s/register?invite=%s", baseURL, code)

	s.log.Info("Invite code created",

		zap.String("code", code),

		zap.String("role", role),

		zap.Int("max_uses", int(maxUses)),
	)

	return &pb.AdminCreateInviteResponse{

		Code: code,

		Role: role,

		Specialty: req.GetSpecialty(),

		MaxUses: maxUses,

		InviteUrl: inviteURL,

		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil

}

func (s *userServer) AdminRevokeInvite(ctx context.Context, req *pb.AdminRevokeInviteRequest) (*pb.AdminRevokeInviteResponse, error) {

	if err := s.requireAdminRole(ctx, ""); err != nil {

		return nil, err

	}

	if req.GetCode() == "" {

		return nil, status.Error(codes.InvalidArgument, "code is required")

	}

	if err := s.inviteCodeRepo.Revoke(ctx, req.GetCode()); err != nil {

		s.log.Error("Failed to revoke invite", zap.Error(err))

		return nil, status.Error(codes.Internal, "failed to revoke invite")

	}

	s.log.Info("Invite code revoked", zap.String("code", req.GetCode()))

	return &pb.AdminRevokeInviteResponse{Success: true, Message: "Invite revoked successfully"}, nil

}

func (s *userServer) AdminDeleteUser(ctx context.Context, req *pb.AdminDeleteUserRequest) (*pb.AdminDeleteUserResponse, error) {

	if err := s.requireAdminRole(ctx, req.RequesterUserId); err != nil {

		return nil, err

	}

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id is required")

	}

	if err := s.userRepo.Delete(ctx, req.UserId); err != nil {

		s.log.Error("Failed to delete user", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to delete user")

	}

	if s.auditLogger != nil {

		s.auditLogger.Log(audit.NewEvent("user-service", audit.EventTypeAdminAction, "admin_delete_user", "user").
			WithUser(req.UserId).
			WithActor(req.RequesterUserId))

	}

	s.log.Info("User deleted", zap.String("user_id", req.UserId))

	return &pb.AdminDeleteUserResponse{Success: true, Message: "User deleted successfully"}, nil

}

func (s *userServer) AdminBanUser(ctx context.Context, req *pb.AdminBanUserRequest) (*pb.AdminBanUserResponse, error) {

	if err := s.requireAdminRole(ctx, req.RequesterUserId); err != nil {

		return nil, err

	}

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id is required")

	}

	if err := s.userRepo.BanUser(ctx, req.UserId); err != nil {

		s.log.Error("Failed to ban user", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to ban user")

	}

	s.log.Info("User banned", zap.String("user_id", req.UserId))

	return &pb.AdminBanUserResponse{Success: true, Message: "User banned successfully"}, nil

}

func (s *userServer) AdminUnbanUser(ctx context.Context, req *pb.AdminUnbanUserRequest) (*pb.AdminUnbanUserResponse, error) {

	if err := s.requireAdminRole(ctx, req.RequesterUserId); err != nil {

		return nil, err

	}

	if req.UserId == "" {

		return nil, status.Error(codes.InvalidArgument, "user_id is required")

	}

	if err := s.userRepo.UnbanUser(ctx, req.UserId); err != nil {

		s.log.Error("Failed to unban user", zap.Error(err), zap.String("user_id", req.UserId))

		return nil, status.Error(codes.Internal, "failed to unban user")

	}

	s.log.Info("User unbanned", zap.String("user_id", req.UserId))

	return &pb.AdminUnbanUserResponse{Success: true, Message: "User unbanned successfully"}, nil

}

type userServerConfig struct {
	database *sql.DB

	log *logger.Logger

	auditLogger *audit.ConsoleAuditLogger

	tokenProvider ports.TokenProvider

	baseURL string

	googleClientID string

	emailSender email.EmailSender

	totpService *totp.Service

	userSvc service.UserService

	userRepo *postgres.PgsodiumUserRepository

	profileRepo port.ProfileRepository

	emailVerifRepo port.EmailVerificationRepository

	refreshTokenRepo port.RefreshTokenRepository

	achievementExRepo port.AchievementRepositoryEx

	inviteCodeRepo port.InviteCodeRepository

	userHealthRepo port.UserHealthConditionRepository

	userBodyCompRepo port.UserBodyCompositionRepository

	userMenstrualRepo port.UserMenstrualRepository

	deviceRepo port.DeviceRepository

	externalSecretsRepo port.ExternalSecretRepository
}

func buildUserServer(cfg userServerConfig) *userServer {

	return &userServer{

		db: cfg.database,

		userSvc: cfg.userSvc,

		userRepo: cfg.userRepo,

		profileRepo: cfg.profileRepo,

		emailVerifRepo: cfg.emailVerifRepo,

		refreshTokenRepo: cfg.refreshTokenRepo,

		achievementExRepo: cfg.achievementExRepo,

		inviteCodeRepo: cfg.inviteCodeRepo,

		userHealthRepo: cfg.userHealthRepo,

		userBodyCompRepo: cfg.userBodyCompRepo,

		userMenstrualRepo: cfg.userMenstrualRepo,

		deviceRepo: cfg.deviceRepo,

		log: cfg.log,

		auditLogger: cfg.auditLogger,

		tokenProvider: cfg.tokenProvider,

		emailSender: cfg.emailSender,

		baseURL: cfg.baseURL,

		googleClientID: cfg.googleClientID,

		totpService: cfg.totpService,
	}

}

func createMetricsServer(metricsPort string) *http.Server {

	metricsMux := http.NewServeMux()

	metricsMux.Handle("/metrics", promhttp.Handler())

	return &http.Server{

		Addr: ":" + metricsPort,

		Handler: metricsMux,

		ReadHeaderTimeout: 5 * time.Second,
	}

}

func connectDatabase(dbCfg db.Config, log *logger.Logger) *sql.DB {

	database, err := db.NewConnection(dbCfg)

	if err != nil {

		log.Fatal("Failed to connect to database", zap.Error(err))

	}

	return database

}

func setupGRPCServer(log *logger.Logger, svc *userServer) *grpc.Server {

	serverOpts := []grpc.ServerOption{grpc.ChainUnaryInterceptor(

		middleware.RecoveryGRPC(log.Logger),

		middleware.CorrelationIDGRPC(),

		metrics.UnaryServerInterceptor("user-service"),
	), telemetry.ServerHandlerOption()}

	s := grpctls.NewServer(serverOpts...)

	pb.RegisterUserServiceServer(s, svc)

	healthServer := health.NewServer()

	grpc_health_v1.RegisterHealthServer(s, healthServer)

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	healthServer.SetServingStatus("user.UserService", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(s)

	return s

}

func initializeUserService(ctx context.Context, log *logger.Logger, database *sql.DB) *userServer {

	jwtPrivateKeyPEM := config.GetEnv("JWT_PRIVATE_KEY_PEM")

	if jwtPrivateKeyPEM == "" {

		log.Fatal("JWT_PRIVATE_KEY_PEM environment variable is required")

	}

	jwtPublicKeyPEM := config.GetEnv("JWT_PUBLIC_KEY_PEM")

	if jwtPublicKeyPEM == "" {

		log.Fatal("JWT_PUBLIC_KEY_PEM environment variable is required")

	}

	totpEncryptionKey := config.GetEnv("TOTP_ENCRYPTION_KEY")

	if totpEncryptionKey == "" {

		log.Fatal("TOTP_ENCRYPTION_KEY environment variable is required")

	}

	totpEncryptor, initErr := crypto.NewAESGCMEncryptor(totpEncryptionKey)

	if initErr != nil {

		log.Fatal("Failed to initialize TOTP encryption", zap.Error(initErr))

	}

	emailCfg := email.LoadConfig()

	emailSender := email.NewSMTPClient(emailCfg)

	baseURL := config.GetEnv("BASE_URL", "https://localhost:8443")

	googleClientID := config.GetEnv("GOOGLE_CLIENT_ID")

	if googleClientID == "" {

		log.Fatal("GOOGLE_CLIENT_ID environment variable is required for Google OAuth")

	}

	tokenProvider := jwt.NewJWTAdapter(jwtPrivateKeyPEM, jwtPublicKeyPEM)

	deviceRepo := postgres.NewDeviceRepository(database)

	userRepo := postgres.NewPgsodiumUserRepository(database)

	profileRepo := postgres.NewProfileRepository(database)

	inviteRepo := postgres.NewInviteRepository(database)

	inviteCodeRepo := postgres.NewInviteCodeRepository(database)

	healthRepo := postgres.NewHealthConditionRepository(database)

	userHealthRepo := postgres.NewUserHealthConditionRepository(database)

	bodyCompRepo := postgres.NewBodyCompositionRepository(database)

	userBodyCompRepo := postgres.NewUserBodyCompositionRepository(database)

	menstrualRepo := postgres.NewMenstrualCycleRepository(database)

	userMenstrualRepo := postgres.NewUserMenstrualRepository(database)

	achievementRepo := postgres.NewAchievementRepository(database)

	achievementExRepo := postgres.NewAchievementRepositoryEx(database)

	emailVerifRepo := postgres.NewEmailVerificationRepository(database)

	refreshTokenRepo := postgres.NewRefreshTokenRepository(database)

	externalSecretsRepo := postgres.NewExternalSecretsRepository(database)

	userSvc := service.NewUserService(service.UserServiceConfig{

		Users: userRepo,

		Profiles: profileRepo,

		Invites: inviteRepo,

		InviteCodes: inviteCodeRepo,

		Health: healthRepo,

		UserHealth: userHealthRepo,

		BodyComp: bodyCompRepo,

		UserBodyComp: userBodyCompRepo,

		Menstrual: menstrualRepo,

		UserMenstrual: userMenstrualRepo,

		Achievements: achievementRepo,

		AchievementsEx: achievementExRepo,

		Devices: deviceRepo,

		EmailVerifs: emailVerifRepo,

		RefreshTokens: refreshTokenRepo,

		ExternalSecrets: externalSecretsRepo,
	})

	svc := buildUserServer(userServerConfig{

		database: database,

		log: log,

		auditLogger: audit.NewConsoleAuditLogger(log.Logger),

		tokenProvider: tokenProvider,

		baseURL: baseURL,

		googleClientID: googleClientID,

		emailSender: emailSender,

		totpService: totp.NewService(totpEncryptor),

		userSvc: userSvc,

		userRepo: userRepo,

		profileRepo: profileRepo,

		emailVerifRepo: emailVerifRepo,

		refreshTokenRepo: refreshTokenRepo,

		achievementExRepo: achievementExRepo,

		inviteCodeRepo: inviteCodeRepo,

		userHealthRepo: userHealthRepo,

		userBodyCompRepo: userBodyCompRepo,

		userMenstrualRepo: userMenstrualRepo,

		deviceRepo: deviceRepo,

		externalSecretsRepo: externalSecretsRepo,
	})
	if err := ensurePgsodiumKey(ctx, database, log); err != nil {

		log.Fatal("Failed to initialize pgsodium keyring", zap.Error(err))

	}

	reencryptPIIFromPgcrypto(ctx, database, log)

	backfillEncryptedPII(ctx, database, log)

	return svc

}

func main() {

	log := logger.New("user-service")

	defer func() { _ = log.Sync() }()

	shutdownTraces := telemetry.InitTracer()

	defer func() {

		if err := shutdownTraces(context.Background()); err != nil {

			log.Warn("Failed to shutdown traces", zap.Error(err))

		}

	}()

	port := config.GetEnv("USER_SERVICE_PORT", "50051")

	metricsPort := config.GetEnv("USER_SERVICE_METRICS_PORT", "9096")

	metricsSrv := createMetricsServer(metricsPort)

	dbCfg := db.Config{

		Host: config.GetEnv("DB_HOST", "localhost"),

		Port: config.GetEnv("DB_PORT", "5432"),

		User: config.GetEnv("POSTGRES_USER"),

		Password: config.GetEnv("POSTGRES_PASSWORD"),

		DBName: config.GetEnv("POSTGRES_DB", "fitness"),

		SSLMode: config.GetEnv("DB_SSLMODE", "disable"),
	}

	database := connectDatabase(dbCfg, log)

	defer func() {

		if closeErr := database.Close(); closeErr != nil {

			log.Error("Failed to close database connection", zap.Error(closeErr))

		}

	}()

	svc := initializeUserService(context.Background(), log, database)

	lc := net.ListenConfig{}

	lis, err := lc.Listen(context.Background(), "tcp", ":"+port)

	if err != nil {

		log.Fatal("Failed to listen", zap.Error(err))

	}

	s := setupGRPCServer(log, svc)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	go func() {

		log.Info("Starting metrics server", zap.String("port", metricsPort))

		if err := metricsSrv.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {

			log.Fatal("Metrics server failed", zap.Error(err))

		}

	}()

	go func() {

		log.Info("User service starting", zap.String("port", port))

		if err := s.Serve(lis); err != nil && !strings.Contains(err.Error(), "Server closed") {

			log.Fatal("Failed to serve", zap.Error(err))

		}

	}()

	<-ctx.Done()

	log.Info("Shutting down user service")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {

		defer wg.Done()

		s.GracefulStop()

	}()

	go func() {

		defer wg.Done()

		_ = metricsSrv.Shutdown(shutdownCtx)

	}()

	wg.Wait()

	log.Info("User service stopped")

}

func generateInviteCode() string {

	b := make([]byte, 6)

	_, _ = rand.Read(b)

	return base64.URLEncoding.EncodeToString(b)[:8]

}
