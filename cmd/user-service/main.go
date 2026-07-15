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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/argon2"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/auth"
	"github.com/MAMUER/project/internal/config"
	"github.com/MAMUER/project/internal/crypto"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/email"
	grpctls "github.com/MAMUER/project/internal/grpc"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/MAMUER/project/internal/telemetry"
	"github.com/MAMUER/project/internal/totp"
	"github.com/MAMUER/project/internal/validator"
)

// User represents a user for login operations.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
}

type userServer struct {
	pb.UnimplementedUserServiceServer
	db               *sql.DB
	log              *logger.Logger
	jwtPrivateKeyPEM string
	emailSender      *email.Sender
	baseURL          string
	googleClientID   string
	totpService      *totp.Service
}

const argon2idParams = "m=65536,t=3,p=1"

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
	expectedLen := 32
	if len(parts[5]) > expectedLen {
		return false
	}
	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
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

func (s *userServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.log.Info("Register request", zap.String("email", req.Email))

	if err := validator.ValidateRegisterRequest(req); err != nil {
		s.log.Warn("Invalid register request", zap.Error(err))
		return nil, fmt.Errorf("validate register request: %w", err)
	}

	email := sanitize.String(req.Email)
	fullName := sanitize.String(req.FullName)
	emailHash := db.EmailHash(email)
	fullNameHash := db.BlindIndex(fullName)
	fullNameNonce, err := db.GenerateNonce()
	if err != nil {
		s.log.Error("Failed to generate nonce", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate nonce")
	}

	var exists bool
	if queryErr := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email_hash = $1)", emailHash).Scan(&exists); queryErr != nil {
		s.log.Error("Database error checking user existence", zap.Error(queryErr))
		return nil, status.Error(codes.Internal, "database error")
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
	userQuery, userArgs := buildUserInsertQuery(userID, email, emailHash, string(hashed), fullName, fullNameNonce, fullNameHash, req.Role)
	if _, execErr := s.db.ExecContext(ctx, userQuery, userArgs...); execErr != nil {
		s.log.Error("Failed to create user", zap.Error(execErr))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	verificationToken := generateVerificationToken()
	verificationQuery, verificationArgs := buildEmailVerificationInsertQuery(userID, email, emailHash, verificationToken)
	if _, execErr := s.db.ExecContext(ctx, verificationQuery, verificationArgs...); execErr != nil {
		s.log.Error("Failed to create email verification record", zap.Error(execErr))
		return nil, status.Error(codes.Internal, "failed to create verification token")
	}

	sendVerificationEmailIfNeeded(s, email, verificationToken)

	if _, profileErr := s.db.ExecContext(ctx, `INSERT INTO user_profiles (user_id) VALUES ($1)`, userID); profileErr != nil {
		s.log.Warn("Failed to create user profile, user will need to complete profile manually",
			zap.Error(profileErr),
			zap.String("user_id", userID))
	}

	return &pb.RegisterResponse{
		UserId:  userID,
		Message: "user created successfully. Verification token (dev only): " + verificationToken,
	}, nil
}

func buildUserInsertQuery(userID, email, emailHash, passwordHash, fullName string, fullNameNonce []byte, fullNameHash, role string) (string, []interface{}) {
	var b strings.Builder
	b.WriteString("INSERT INTO users (id, email_encrypted, email_hash, password_hash, full_name_encrypted, full_name_nonce, full_name_hash, role, created_at, updated_at) ")
	b.WriteString("VALUES ($1, ")
	b.WriteString(db.PgsodiumEncryptParam(2))
	b.WriteString(", $3, $4, ")
	b.WriteString(db.PgsodiumRandomEncryptParam(5, 6))
	b.WriteString(", $7, ")
	b.WriteString("$8, ")
	b.WriteString("$9, NOW(), NOW())")
	args := []interface{}{userID, email, emailHash, passwordHash, fullName, fullNameNonce, fullNameHash, role}
	return b.String(), args
}

func buildEmailVerificationInsertQuery(userID, email, emailHash, verificationToken string) (string, []interface{}) {
	var b strings.Builder
	b.WriteString("INSERT INTO email_verifications (user_id, email_encrypted, email_hash, token, token_encrypted, expires_at, used) ")
	b.WriteString("VALUES ($1, ")
	b.WriteString(db.PgsodiumEncryptParam(2))
	b.WriteString(", $3, $4, ")
	b.WriteString(db.PgsodiumEncryptParam(5))
	b.WriteString(", NOW() + INTERVAL '24 hours', false)")
	args := []interface{}{userID, email, emailHash, verificationToken}
	return b.String(), args
}

func sendVerificationEmailIfNeeded(s *userServer, email, verificationToken string) {
	if s.emailSender == nil || s.baseURL == "" {
		return
	}
	if sendErr := s.emailSender.SendVerificationEmail(email, verificationToken, s.baseURL); sendErr != nil {
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

	// Ищем запись о верификации
	var userID, email string
	var used bool
	var expiresAt sql.NullTime
	var confirmEmailQuery strings.Builder
	confirmEmailQuery.WriteString("SELECT user_id, ")
	confirmEmailQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email"))
	confirmEmailQuery.WriteString("\n        FROM email_verifications \n        WHERE token = $1")
	err := s.db.QueryRowContext(ctx, confirmEmailQuery.String(), req.Token).Scan(&userID, &email, &used, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.InvalidArgument, "invalid verification token")
	}
	if err != nil {
		s.log.Error("Database error checking verification token", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	// Проверяем, не использован ли токен
	if used {
		return nil, status.Error(codes.InvalidArgument, "verification token has already been used")
	}

	// Проверяем, не истёк ли токен
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return nil, status.Error(codes.InvalidArgument, "verification token has expired")
	}

	// Обновляем: помечаем токен как использованный и подтверждаем email
	_, err = s.db.ExecContext(ctx, `
        UPDATE email_verifications SET used = true WHERE token = $1
    `, req.Token)
	if err != nil {
		s.log.Error("Failed to update verification token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to confirm email")
	}

	_, err = s.db.ExecContext(ctx, `
        UPDATE users SET email_confirmed = true WHERE id = $1
    `, userID)
	if err != nil {
		s.log.Error("Failed to update user email_confirmed", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to confirm email")
	}

	s.log.Info("Email confirmed", zap.String("user_id", userID), zap.String("email", email))
	return &pb.ConfirmEmailResponse{
		UserId:  userID,
		Message: "email confirmed successfully",
	}, nil
}

func (s *userServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	s.log.Info("Login request", zap.String("email", req.Email))

	// Валидация входных данных
	if err := validator.ValidateLoginRequest(req); err != nil {
		s.log.Warn("Invalid login request", zap.Error(err))
		return nil, fmt.Errorf("validate login request: %w", err)
	}

	// Проверка подтверждения email
	emailHash := db.EmailHash(sanitize.String(req.Email))
	var emailConfirmed bool
	err := s.db.QueryRowContext(ctx, "SELECT email_confirmed FROM users WHERE email_hash = $1", emailHash).Scan(&emailConfirmed)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		s.log.Error("Database error checking email confirmation", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	if !emailConfirmed {
		s.log.Info("Login attempt with unconfirmed email", zap.String("email", req.Email))
		return nil, status.Error(codes.Unauthenticated, "Email not confirmed. Please check your inbox.")
	}

	var user User
	var loginQuery strings.Builder
	loginQuery.WriteString("SELECT id, ")
	loginQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email"))
	loginQuery.WriteString(", password_hash, role \n        FROM users \n        WHERE email_hash = $1")
	err = s.db.QueryRowContext(ctx, loginQuery.String(), emailHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		// Возвращаем Unauthenticated вместо NotFound для безопасности
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		s.log.Error("Database error during login", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	// Проверка пароля
	if !verifyPasswordArgon2id(user.PasswordHash, req.Password) {
		s.log.Info("Invalid login attempt", zap.String("email", req.Email))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Генерация JWT
	token, err := auth.GenerateAccessToken(user.ID, user.Email, user.Role, s.jwtPrivateKeyPEM, 15*time.Minute)
	if err != nil {
		s.log.Error("Failed to generate JWT", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   15 * 60,
		UserId:      user.ID,
		Role:        user.Role,
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

	token, tokenErr := auth.GenerateAccessToken(userID, emailVal, role, s.jwtPrivateKeyPEM, 15*time.Minute)
	if tokenErr != nil {
		s.log.Error("Failed to generate JWT", zap.Error(tokenErr))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	s.log.Info("Google auth successful", zap.String("user_id", userID), zap.String("email", emailVal))
	return &pb.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   15 * 60,
		UserId:      userID,
		Role:        role,
	}, nil
}

func (s *userServer) findOrCreateGoogleUser(ctx context.Context, googleSub, emailHash, emailVal string) (userID, role string, emailConfirmed bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT id, role, email_confirmed FROM users WHERE provider = 'google' AND external_id = $1
	`, googleSub).Scan(&userID, &role, &emailConfirmed)
	if err == nil {
		if !emailConfirmed {
			_, _ = s.db.ExecContext(ctx, `UPDATE users SET email_confirmed = true, updated_at = NOW() WHERE id = $1`, userID)
			emailConfirmed = true
		}
		return userID, role, emailConfirmed, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("Database error during Google auth", zap.Error(err))
		return "", "", false, status.Error(codes.Internal, "database error")
	}

	err = s.db.QueryRowContext(ctx, `
		SELECT id, role, email_confirmed FROM users WHERE email_hash = $1
	`, emailHash).Scan(&userID, &role, &emailConfirmed)
	if err == nil {
		_, linkErr := s.db.ExecContext(ctx, `
			UPDATE users SET provider = 'google', external_id = $1, email_confirmed = true, updated_at = NOW() WHERE id = $2
		`, googleSub, userID)
		if linkErr != nil {
			s.log.Warn("Failed to link Google account", zap.Error(linkErr), zap.String("user_id", userID))
		}
		return userID, role, emailConfirmed, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		s.log.Error("Database error during Google auth", zap.Error(err))
		return "", "", false, status.Error(codes.Internal, "database error")
	}

	nickname := extractLocalPart(emailVal)
	nicknameHash := db.NicknameHash(nickname)
	nicknameNonce, err := db.GenerateNonce()
	if err != nil {
		s.log.Error("Failed to generate nonce", zap.Error(err))
		return "", "", false, status.Error(codes.Internal, "failed to generate nonce")
	}
	userID = uuid.New().String()
	var b strings.Builder
	b.WriteString("INSERT INTO users (id, email_encrypted, email_hash, password_hash, nickname_encrypted, nickname_nonce, nickname_hash, role, provider, external_id, email_confirmed, created_at, updated_at) ")
	b.WriteString("VALUES ($1, ")
	b.WriteString(db.PgsodiumEncryptParam(2))
	b.WriteString(", $3, NULL, ")
	b.WriteString(db.PgsodiumRandomEncryptParam(4, 5))
	b.WriteString(", $6, $7, ")
	b.WriteString("'client', 'google', $8, true, NOW(), NOW())")
	_, insertErr := s.db.ExecContext(ctx, b.String(), userID, emailVal, emailHash, nickname, nicknameNonce, nicknameHash, googleSub)
	if insertErr != nil {
		var pqErr *pq.Error
		if errors.As(insertErr, &pqErr) && pqErr.Code == "23505" {
			return "", "", false, status.Error(codes.AlreadyExists, "user already exists")
		}
		s.log.Error("Failed to create OAuth user", zap.Error(insertErr))
		return "", "", false, status.Error(codes.Internal, "failed to create user")
	}
	role = "client"
	emailConfirmed = true

	_, profileErr := s.db.ExecContext(ctx, `INSERT INTO user_profiles (user_id) VALUES ($1)`, userID)
	if profileErr != nil {
		s.log.Warn("Failed to create profile for OAuth user", zap.Error(profileErr), zap.String("user_id", userID))
	}
	return userID, role, emailConfirmed, nil
}

func (s *userServer) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.UserProfile, error) {
	var profile pb.UserProfile
	var age sql.NullInt32
	var gender sql.NullString
	var heightCm sql.NullInt32
	var weightKg sql.NullFloat64
	var fitnessLevel sql.NullString
	var nutrition sql.NullString
	var sleepHours sql.NullFloat64

	var nickname, profilePhotoURL sql.NullString

	var err error
	if db.PgsodiumKeyID() > 0 {
		var getProfileQuery strings.Builder
		getProfileQuery.WriteString("SELECT u.id, ")
		getProfileQuery.WriteString(db.PgsodiumDecryptParam("u.email_encrypted", "email"))
		getProfileQuery.WriteString(",\n               ")
		getProfileQuery.WriteString(db.PgsodiumDecryptDualParam("u.full_name_encrypted", "u.full_name_nonce", "full_name"))
		getProfileQuery.WriteString(",\n               ")
		getProfileQuery.WriteString(db.PgsodiumDecryptDualParam("u.nickname_encrypted", "u.nickname_nonce", "nickname"))
		getProfileQuery.WriteString(",\n               u.profile_photo_url, u.role,\n               p.age, p.gender, p.height_cm, p.weight_kg, p.fitness_level,\n               p.goals, p.nutrition, p.sleep_hours,\n               u.created_at, u.updated_at\n            FROM users u\n            LEFT JOIN user_profiles_with_goals p ON u.id = p.user_id\n            WHERE u.id = $1")
		err = s.db.QueryRowContext(ctx, getProfileQuery.String(), req.UserId).Scan(
			&profile.UserId, &profile.Email, &profile.FullName, &nickname, &profilePhotoURL, &profile.Role,
			&age, &gender, &heightCm, &weightKg, &fitnessLevel,
			pq.Array(&profile.Goals),
			&nutrition, &sleepHours,
			&profile.CreatedAt, &profile.UpdatedAt,
		)
	} else {
		err = s.db.QueryRowContext(ctx, `
			SELECT u.id, u.email, u.full_name, u.nickname, u.profile_photo_url, u.role,
			       p.age, p.gender, p.height_cm, p.weight_kg, p.fitness_level,
			       p.goals, p.nutrition, p.sleep_hours,
			       u.created_at, u.updated_at
			FROM users u
			LEFT JOIN user_profiles_with_goals p ON u.id = p.user_id
			WHERE u.id = $1
		`, req.UserId).Scan(
			&profile.UserId, &profile.Email, &profile.FullName, &nickname, &profilePhotoURL, &profile.Role,
			&age, &gender, &heightCm, &weightKg, &fitnessLevel,
			pq.Array(&profile.Goals),
			&nutrition, &sleepHours,
			&profile.CreatedAt, &profile.UpdatedAt,
		)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		s.log.Error("Database error getting profile", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "database error")
	}

	if nickname.Valid {
		profile.Nickname = nickname.String
	}
	if profilePhotoURL.Valid {
		profile.ProfilePhotoUrl = profilePhotoURL.String
	}
	if age.Valid {
		profile.Age = age.Int32
	}
	if gender.Valid {
		profile.Gender = gender.String
	}
	if heightCm.Valid {
		profile.HeightCm = heightCm.Int32
	}
	if weightKg.Valid {
		profile.WeightKg = weightKg.Float64
	}
	if fitnessLevel.Valid {
		profile.FitnessLevel = fitnessLevel.String
	}
	if nutrition.Valid {
		profile.Nutrition = nutrition.String
	}
	if sleepHours.Valid {
		profile.SleepHours = float32(sleepHours.Float64)
	}

	return &profile, nil
}

func (s *userServer) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.UserProfile, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	emailHash := db.EmailHash(req.Email)
	var profile pb.UserProfile
	var emailConfirmed bool
	var createdAt, updatedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, email_confirmed, created_at, updated_at
		FROM users
		WHERE email_hash = $1
	`, emailHash).Scan(&profile.UserId, &emailConfirmed, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		s.log.Error("Database error getting user by email", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	profile.EmailConfirmed = emailConfirmed
	if createdAt.Valid {
		profile.CreatedAt = createdAt.Time.Format(time.RFC3339)
	}
	if updatedAt.Valid {
		profile.UpdatedAt = updatedAt.Time.Format(time.RFC3339)
	}

	if db.PgsodiumKeyID() > 0 {
		var email string
		emailQuery := strings.Builder{}
		emailQuery.WriteString("SELECT ")
		emailQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email"))
		emailQuery.WriteString(" FROM users WHERE id = $1")
		if err := s.db.QueryRowContext(ctx, emailQuery.String(), profile.UserId).Scan(&email); err != nil {
			s.log.Error("Failed to decrypt email", zap.Error(err))
			return nil, status.Error(codes.Internal, "failed to decrypt email")
		}
		profile.Email = email
	}

	return &profile, nil
}

func (s *userServer) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh_token is required")
	}

	var userID string
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at FROM refresh_tokens WHERE token = $1 AND used = FALSE
	`, req.RefreshToken).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if err != nil {
		s.log.Error("Database error checking refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	if expiresAt.Before(time.Now()) {
		return nil, status.Error(codes.Unauthenticated, "refresh token expired")
	}

	_, err = s.db.ExecContext(ctx, `UPDATE refresh_tokens SET used = TRUE WHERE token = $1`, req.RefreshToken)
	if err != nil {
		s.log.Error("Failed to mark refresh token as used", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	var email, role string
	if err := s.db.QueryRowContext(ctx, `SELECT email, role FROM users WHERE id = $1`, userID).Scan(&email, &role); err != nil {
		s.log.Error("Failed to get user for refresh", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	accessToken, err := auth.GenerateAccessToken(userID, email, role, s.jwtPrivateKeyPEM, 15*time.Minute)
	if err != nil {
		s.log.Error("Failed to generate access token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	newRefresh := auth.GenerateRefreshToken()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)
	`, userID, newRefresh, time.Now().Add(7*24*time.Hour)); err != nil {
		s.log.Error("Failed to store new refresh token", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	return &pb.RefreshTokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    900,
		UserId:       userID,
		Role:         role,
		RefreshToken: newRefresh,
	}, nil
}

func (s *userServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserProfile, error) {
	if err := validator.ValidateProfileUpdate(req); err != nil {
		s.log.Warn("Invalid profile update request", zap.Error(err))
		return nil, fmt.Errorf("validate profile update: %w", err)
	}

	// Обновляем full_name и nickname в users table (если передан)
	if req.FullName != nil || req.Nickname != nil {
		fullNameNonce, _ := db.GenerateNonce()
		fullNameHash := db.BlindIndex(toString(req.FullName))
		nicknameNonce, _ := db.GenerateNonce()
		nicknameHash := db.BlindIndex(toString(req.Nickname))
		var updateProfileQuery strings.Builder
		updateProfileQuery.WriteString("UPDATE users SET\n\t\t\t\tfull_name_encrypted = CASE WHEN $1 IS NULL THEN full_name_encrypted ELSE ")
		updateProfileQuery.WriteString(db.PgsodiumRandomEncryptParam(1, 2))
		updateProfileQuery.WriteString(" END,\n\t\t\t\tfull_name_nonce = CASE WHEN $1 IS NULL THEN full_name_nonce ELSE $2 END,\n\t\t\t\tfull_name_hash = CASE WHEN $1 IS NULL THEN full_name_hash ELSE $3 END,\n\t\t\t\tnickname_encrypted = CASE WHEN $4 IS NULL THEN nickname_encrypted ELSE ")
		updateProfileQuery.WriteString(db.PgsodiumRandomEncryptParam(4, 5))
		updateProfileQuery.WriteString(" END,\n\t\t\t\tnickname_nonce = CASE WHEN $4 IS NULL THEN nickname_nonce ELSE $5 END,\n\t\t\t\tnickname_hash = CASE WHEN $4 IS NULL THEN nickname_hash ELSE $6 END,\n\t\t\t\tupdated_at = NOW()\n\t\t\tWHERE id = $7")
		_, err := s.db.ExecContext(ctx, updateProfileQuery.String(), toString(req.FullName), fullNameNonce, fullNameHash, toString(req.Nickname), nicknameNonce, nicknameHash, req.UserId)
		if err != nil {
			s.log.Error("Failed to update user details", zap.Error(err), zap.String("user_id", req.UserId))
			return nil, status.Error(codes.Internal, "failed to update user details")
		}
	}

	// Обновляем user_profiles (без goals/contraindications — они в отдельных таблицах)
	// Сначала проверяем что пользователь существует
	var userExists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", req.UserId).Scan(&userExists)
	if err != nil {
		s.log.Error("Failed to check user existence", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "database error")
	}
	if !userExists {
		s.log.Error("User not found during profile update", zap.String("user_id", req.UserId))
		return nil, status.Error(codes.NotFound, "user not found")
	}

	profileQuery := `
        INSERT INTO user_profiles (user_id, age, gender, height_cm, weight_kg, fitness_level, nutrition, sleep_hours, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
        ON CONFLICT (user_id) DO UPDATE SET
            age = COALESCE(EXCLUDED.age, user_profiles.age),
            gender = COALESCE(EXCLUDED.gender, user_profiles.gender),
            height_cm = COALESCE(EXCLUDED.height_cm, user_profiles.height_cm),
            weight_kg = COALESCE(EXCLUDED.weight_kg, user_profiles.weight_kg),
            fitness_level = COALESCE(EXCLUDED.fitness_level, user_profiles.fitness_level),
            nutrition = COALESCE(EXCLUDED.nutrition, user_profiles.nutrition),
            sleep_hours = COALESCE(EXCLUDED.sleep_hours, user_profiles.sleep_hours),
            updated_at = NOW()
    `

	_, err = s.db.ExecContext(ctx, profileQuery,
		req.UserId,
		req.Age, req.Gender, req.HeightCm, req.WeightKg, req.FitnessLevel,
		req.Nutrition, req.SleepHours,
	)
	if err != nil {
		s.log.Error("Failed to update profile", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to update profile")
	}

	// Обновляем goals
	if err := s.updateUserList(ctx, req.UserId, "user_goals", "goal", req.Goals, "goals"); err != nil {
		return nil, err
	}

	// Обновляем contraindications
	if err := s.updateUserList(ctx, req.UserId, "user_contraindications", "contraindication", req.Contraindications, "contraindications"); err != nil {
		return nil, err
	}

	// Возвращаем обновленный профиль
	return s.GetProfile(ctx, &pb.GetProfileRequest{UserId: req.UserId})
}

func (s *userServer) updateUserList(ctx context.Context, userID, tableName, columnName string, items []string, logMsg string) error {
	if len(items) == 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE user_id = $1`, tableName), userID)
	if err != nil {
		s.log.Error("Failed to delete old "+logMsg, zap.Error(err), zap.String("user_id", userID))
		return status.Errorf(codes.Internal, "failed to update %s", logMsg)
	}
	for _, item := range items {
		_, err = s.db.ExecContext(ctx,
			fmt.Sprintf(`INSERT INTO %s (user_id, %s) VALUES ($1, $2) ON CONFLICT DO NOTHING`, tableName, columnName),
			userID, item)
		if err != nil {
			s.log.Error("Failed to insert "+logMsg, zap.Error(err), zap.String("user_id", userID))
			return status.Errorf(codes.Internal, "failed to update %s", logMsg)
		}
	}
	return nil
}

func (s *userServer) deleteRecord(ctx context.Context, tableName, idField, userID, recordID, logMsg string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE %s = $1 AND user_id = $2`, tableName, idField), recordID, userID)
	if err != nil {
		s.log.Error("Failed to delete "+logMsg, zap.Error(err))
		return status.Error(codes.Internal, "database error")
	}
	return nil
}

// ChangePassword changes the user's password after verifying the current one.
func (s *userServer) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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
	var currentHash string
	err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = $1", req.UserId).Scan(&currentHash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		s.log.Error("Failed to fetch password hash", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "database error")
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
	_, err = s.db.ExecContext(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(newHash), req.UserId)
	if err != nil {
		s.log.Error("Failed to update password", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to update password")
	}

	s.log.Info("Password changed successfully", zap.String("user_id", req.UserId))
	return &pb.ChangePasswordResponse{Message: "Password changed successfully"}, nil
}

// ChangeNickname changes the user's nickname.
func (s *userServer) ChangeNickname(ctx context.Context, req *pb.ChangeNicknameRequest) (*pb.ChangeNicknameResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.NewNickname == "" {
		return nil, status.Error(codes.InvalidArgument, "new_nickname is required")
	}

	// Check if nickname is unique (check plaintext column for existing plaintext rows)
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE nickname = $1 AND id != $2)", req.NewNickname, req.UserId).Scan(&exists)
	if err != nil {
		s.log.Error("Failed to check nickname uniqueness", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "nickname already taken")
	}

	// Update nickname in both encrypted and plaintext columns
	nicknameHash := db.NicknameHash(req.NewNickname)
	nicknameNonce, err := db.GenerateNonce()
	if err != nil {
		s.log.Error("Failed to generate nonce", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate nonce")
	}
	var nicknameQuery strings.Builder
	nicknameQuery.WriteString("UPDATE users SET nickname_encrypted = ")
	nicknameQuery.WriteString(db.PgsodiumRandomEncryptParam(1, 2))
	nicknameQuery.WriteString(", nickname_nonce = $2, nickname_hash = $3, updated_at = NOW() WHERE id = $4")
	_, err = s.db.ExecContext(ctx, nicknameQuery.String(), req.NewNickname, nicknameNonce, nicknameHash, req.UserId)
	if err != nil {
		s.log.Error("Failed to update nickname", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to update nickname")
	}

	s.log.Info("Nickname changed", zap.String("user_id", req.UserId), zap.String("new_nickname", req.NewNickname))
	return &pb.ChangeNicknameResponse{Message: "Nickname changed successfully"}, nil
}

// UploadProfilePhoto uploads a new profile photo for the user.
func (s *userServer) UploadProfilePhoto(ctx context.Context, req *pb.UploadProfilePhotoRequest) (*pb.UploadProfilePhotoResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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

	_, err := s.db.ExecContext(ctx, "UPDATE users SET profile_photo_url = $1, updated_at = NOW() WHERE id = $2", photoURL, req.UserId)
	if err != nil {
		s.log.Error("Failed to update profile photo URL", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to update profile photo")
	}

	s.log.Info("Profile photo uploaded", zap.String("user_id", req.UserId), zap.String("photo_url", photoURL))
	return &pb.UploadProfilePhotoResponse{PhotoUrl: photoURL}, nil
}

// RemoveProfilePhoto removes the user's profile photo.
func (s *userServer) RemoveProfilePhoto(ctx context.Context, req *pb.RemoveProfilePhotoRequest) (*pb.RemoveProfilePhotoResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Update DB to remove photo URL
	_, err := s.db.ExecContext(ctx, "UPDATE users SET profile_photo_url = NULL, updated_at = NOW() WHERE id = $1", req.UserId)
	if err != nil {
		s.log.Error("Failed to remove profile photo", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to remove profile photo")
	}

	s.log.Info("Profile photo removed", zap.String("user_id", req.UserId))
	return &pb.RemoveProfilePhotoResponse{Message: "Profile photo removed successfully"}, nil
}

// ListDevices lists the user's connected devices.
func (s *userServer) ListDevices(ctx context.Context, req *pb.ListDevicesRequest) (*pb.ListDevicesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Query devices
	rows, err := s.db.QueryContext(ctx, `
		SELECT id as device_id, device_type, device_name, is_connected, last_sync
		FROM devices WHERE user_id = $1
	`, req.UserId)
	if err != nil {
		s.log.Error("Failed to list devices", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to list devices")
	}
	defer func() { _ = rows.Close() }()

	var devices []*pb.Device
	for rows.Next() {
		var d pb.Device
		var lastSync sql.NullString
		err := rows.Scan(&d.DeviceId, &d.DeviceType, &d.DeviceName, &d.IsConnected, &lastSync)
		if err != nil {
			s.log.Error("Failed to scan device", zap.Error(err))
			continue
		}
		if lastSync.Valid {
			d.LastSync = lastSync.String
		}
		devices = append(devices, &d)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "failed to list devices")
	}
	return &pb.ListDevicesResponse{Devices: devices}, nil
}

// AddDevice adds a new device for the user.
func (s *userServer) AddDevice(ctx context.Context, req *pb.AddDeviceRequest) (*pb.AddDeviceResponse, error) {
	if req.UserId == "" || req.DeviceType == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and device_type are required")
	}

	deviceID := uuid.New().String()
	deviceName := req.DeviceName
	if deviceName == "" {
		deviceName = req.DeviceType + " Device"
	}

	// Insert device
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO devices (id, user_id, device_type, device_name, token, is_connected, last_sync)
		VALUES ($1, $2, $3, $4, $5, true, NOW())
	`, deviceID, req.UserId, req.DeviceType, deviceName, uuid.New().String())
	if err != nil {
		s.log.Error("Failed to add device", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to add device")
	}

	device := &pb.Device{
		DeviceId:    deviceID,
		DeviceType:  req.DeviceType,
		DeviceName:  deviceName,
		IsConnected: true,
		LastSync:    time.Now().Format(time.RFC3339),
	}

	s.log.Info("Device added", zap.String("user_id", req.UserId), zap.String("device_id", deviceID))
	return &pb.AddDeviceResponse{Device: device}, nil
}

// RemoveDevice removes a device from the user.
func (s *userServer) RemoveDevice(ctx context.Context, req *pb.RemoveDeviceRequest) (*pb.RemoveDeviceResponse, error) {
	if req.UserId == "" || req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and device_id are required")
	}

	// Delete device
	result, err := s.db.ExecContext(ctx, "DELETE FROM devices WHERE user_id = $1 AND id = $2", req.UserId, req.DeviceId)
	if err != nil {
		s.log.Error("Failed to remove device", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to remove device")
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		return nil, status.Error(codes.NotFound, "device not found")
	}

	s.log.Info("Device removed", zap.String("user_id", req.UserId), zap.String("device_id", req.DeviceId))
	return &pb.RemoveDeviceResponse{Message: "Device removed successfully"}, nil
}

// SyncDeviceData syncs data from the device (stub implementation).
func (s *userServer) SyncDeviceData(ctx context.Context, req *pb.SyncDeviceDataRequest) (*pb.SyncDeviceDataResponse, error) {
	if req.UserId == "" || req.DeviceId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and device_id are required")
	}

	// In real implementation, trigger sync with real API
	// For now, simulate sync by updating last_sync
	_, err := s.db.ExecContext(ctx, "UPDATE devices SET last_sync = NOW() WHERE user_id = $1 AND id = $2", req.UserId, req.DeviceId)
	if err != nil {
		s.log.Error("Failed to sync device data", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to sync device data")
	}

	// Simulate synced samples
	syncedSamples := 100 // Placeholder

	s.log.Info("Device data synced", zap.String("user_id", req.UserId), zap.String("device_id", req.DeviceId), zap.Int("samples", syncedSamples))
	return &pb.SyncDeviceDataResponse{Message: "Device data synced successfully", SyncedSamples: int32(syncedSamples)}, nil
}

// GetTrainingStats retrieves training statistics for the user (stub implementation).
func (s *userServer) GetTrainingStats(ctx context.Context, req *pb.GetTrainingStatsRequest) (*pb.GetTrainingStatsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// In real implementation, query training service or database for stats
	// For now, return mock data
	stats := &pb.TrainingStats{
		TotalWorkouts:          25,
		CompletedWorkouts:      20,
		AverageDurationMinutes: 45.5,
		TotalCaloriesBurned:    1500.0,
		MostFrequentExercise:   "Push-ups",
	}

	return &pb.GetTrainingStatsResponse{Stats: stats}, nil
}

// GetAchievements retrieves all achievements with user's earned status from database.
func (s *userServer) GetAchievements(ctx context.Context, req *pb.GetAchievementsRequest) (*pb.GetAchievementsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.name, a.description, a.icon_url, ua.earned_at
		FROM achievements a
		LEFT JOIN user_achievements ua ON ua.achievement_id = a.id AND ua.user_id = $1
		ORDER BY a.created_at ASC
	`, req.UserId)
	if err != nil {
		s.log.Error("Failed to query achievements", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to query achievements")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Warn("Failed to close achievements rows", zap.Error(closeErr))
		}
	}()

	var achievements []*pb.Achievement
	for rows.Next() {
		var id, name, description, iconURL string
		var earnedAt sql.NullTime
		if err := rows.Scan(&id, &name, &description, &iconURL, &earnedAt); err != nil {
			s.log.Error("Failed to scan achievement", zap.Error(err))
			continue
		}
		earnedDate := ""
		if earnedAt.Valid {
			earnedDate = earnedAt.Time.Format(time.RFC3339)
		}
		achievements = append(achievements, &pb.Achievement{
			AchievementId: id,
			Title:         name,
			Description:   description,
			EarnedDate:    earnedDate,
			IconUrl:       iconURL,
		})
	}

	if err := rows.Err(); err != nil {
		s.log.Error("Achievement rows error", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to read achievements")
	}

	if achievements == nil {
		achievements = []*pb.Achievement{}
	}

	return &pb.GetAchievementsResponse{Achievements: achievements}, nil
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

func (s *userServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	// Валидация параметров
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}
	if req.PageSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "page_size must be greater than 0")
	}
	if req.Page < 0 {
		return nil, status.Error(codes.InvalidArgument, "page must be non-negative")
	}

	offset := req.Page * req.PageSize
	var listUsersQuery strings.Builder
	listUsersQuery.WriteString("SELECT u.id, ")
	listUsersQuery.WriteString(db.PgsodiumDecryptParam("u.email_encrypted", "email"))
	listUsersQuery.WriteString(",\n               ")
	listUsersQuery.WriteString(db.PgsodiumDecryptDualParam("u.full_name_encrypted", "u.full_name_nonce", "full_name"))
	listUsersQuery.WriteString(", u.role, u.created_at, u.updated_at\n        FROM users u\n        WHERE ($1 = '' OR u.role = $1)\n        ORDER BY u.created_at DESC\n        LIMIT $2 OFFSET $3")
	rows, err := s.db.QueryContext(ctx, listUsersQuery.String(), req.Role, req.PageSize, offset)
	if err != nil {
		s.log.Error("Failed to list users", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error("Failed to close rows for body composition", zap.Error(closeErr))
		}
	}()

	var users []*pb.UserProfile
	for rows.Next() {
		var user pb.UserProfile
		if scanErr := rows.Scan(&user.UserId, &user.Email, &user.FullName, &user.Role, &user.CreatedAt, &user.UpdatedAt); scanErr != nil {
			s.log.Error("Failed to scan user", zap.Error(scanErr))
			return nil, status.Error(codes.Internal, "failed to read user data")
		}
		users = append(users, &user)
	}

	// Проверяем ошибку итерации
	if rowErr := rows.Err(); rowErr != nil {
		s.log.Error("Row iteration error", zap.Error(rowErr))
		return nil, status.Error(codes.Internal, "error reading users")
	}

	var total int32
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE ($1 = '' OR role = $1)", req.Role).Scan(&total)
	if err != nil {
		s.log.Warn("Failed to count users", zap.Error(err))
		// Не блокируем ответ, просто логируем
	}

	return &pb.ListUsersResponse{
		Users: users,
		Total: total,
	}, nil
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
	result := s.db.QueryRowContext(ctx, `SELECT * FROM use_invite_code($1)`, req.GetInviteCode())
	var isValid bool
	var role, specialty, errMsg string
	if err := result.Scan(&isValid, &role, &specialty, &errMsg); err != nil {
		s.log.Error("Failed to validate invite code", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !isValid {
		return nil, status.Errorf(codes.InvalidArgument, "invite code error: %s", errMsg)
	}

	// Определяем роль: приоритет у invite_code role
	finalRole := role

	// Хешируем пароль
	hashedPassword, err := hashPasswordArgon2id(req.GetPassword())
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Создаём пользователя
	userID := uuid.New().String()
	emailVal := sanitize.String(req.GetEmail())
	fullName := sanitize.String(req.GetFullName())
	fullNameHash := db.BlindIndex(fullName)
	fullNameNonce, err := db.GenerateNonce()
	if err != nil {
		s.log.Error("Failed to generate nonce", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate nonce")
	}
	emailHash := db.EmailHash(emailVal)
	var registerWithInviteQuery strings.Builder
	registerWithInviteQuery.WriteString("INSERT INTO users (id, email_encrypted, email_hash, password_hash, full_name_encrypted, full_name_nonce, full_name_hash, role, email_confirmed) ")
	registerWithInviteQuery.WriteString("VALUES ($1, ")
	registerWithInviteQuery.WriteString(db.PgsodiumEncryptParam(2))
	registerWithInviteQuery.WriteString(", $3, $4, ")
	registerWithInviteQuery.WriteString(db.PgsodiumRandomEncryptParam(5, 6))
	registerWithInviteQuery.WriteString(", $7, ")
	registerWithInviteQuery.WriteString("$8, true)")
	_, err = s.db.ExecContext(ctx, registerWithInviteQuery.String(), userID, emailVal, emailHash, string(hashedPassword), fullName, fullNameNonce, fullNameHash, finalRole)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		s.log.Error("Failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Генерируем JWT (токен возвращается при login, не при регистрации)
	_, err = auth.GenerateAccessToken(userID, req.GetEmail(), finalRole, s.jwtPrivateKeyPEM, 15*time.Minute)
	if err != nil {
		s.log.Error("Failed to generate JWT", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	s.log.Info("User registered via invite code",
		zap.String("user_id", userID),
		zap.String("email", req.GetEmail()),
		zap.String("role", finalRole),
	)

	return &pb.RegisterResponse{
		UserId:  userID,
		Message: "Регистрация успешна",
	}, nil
}

func (s *userServer) ValidateInviteCode(ctx context.Context, req *pb.ValidateInviteCodeRequest) (*pb.ValidateInviteCodeResponse, error) {
	if req.GetCode() == "" {
		return nil, status.Error(codes.InvalidArgument, "code is required")
	}

	result := s.db.QueryRowContext(ctx, `SELECT * FROM use_invite_code($1)`, req.GetCode())
	var isValid bool
	var role, specialty, errMsg string
	if err := result.Scan(&isValid, &role, &specialty, &errMsg); err != nil {
		s.log.Error("Failed to validate invite code", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &pb.ValidateInviteCodeResponse{
		IsValid:      isValid,
		Role:         role,
		Specialty:    specialty,
		ErrorMessage: errMsg,
	}, nil
}

func (s *userServer) SetupTOTP(ctx context.Context, req *pb.SetupTOTPRequest) (*pb.SetupTOTPResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var email string
	var totpEnabled bool
	var setupTOTPQuery strings.Builder
	setupTOTPQuery.WriteString("SELECT ")
	setupTOTPQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email"))
	setupTOTPQuery.WriteString(", totp_enabled FROM users WHERE id = $1")
	err := s.db.QueryRowContext(ctx, setupTOTPQuery.String(), req.UserId).Scan(&email, &totpEnabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "database error")
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
		QrCodeUrl:   setup.QRCodeURL,
		Secret:      setup.Secret,
		BackupCodes: setup.BackupCodes,
	}, nil
}

func (s *userServer) ConfirmTOTP(ctx context.Context, req *pb.ConfirmTOTPRequest) (*pb.ConfirmTOTPResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
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
		return nil, status.Error(codes.Internal, "failed to validate TOTP code")
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

	_, err = s.db.ExecContext(ctx, `
		UPDATE users 
		SET totp_secret_encrypted = $1, totp_enabled = true, totp_backup_codes_hash = $2, totp_backup_codes_remaining = $3, updated_at = NOW()
		WHERE id = $4
	`, encryptedSecret, pq.Array(hashedCodes), len(req.BackupCodes), req.UserId)
	if err != nil {
		s.log.Error("Failed to enable TOTP", zap.Error(err), zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Internal, "failed to enable TOTP")
	}

	s.log.Info("TOTP enabled successfully", zap.String("user_id", req.UserId))
	return &pb.ConfirmTOTPResponse{Success: true, Message: "TOTP enabled successfully"}, nil
}

func (s *userServer) VerifyTOTP(ctx context.Context, req *pb.VerifyTOTPRequest) (*pb.VerifyTOTPResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Passcode == "" {
		return nil, status.Error(codes.InvalidArgument, "passcode is required")
	}

	var encryptedSecret []byte
	var totpEnabled bool
	var backupCodesHash []string
	var backupCodesRemaining int32

	err := s.db.QueryRowContext(ctx, `
		SELECT totp_secret_encrypted, totp_enabled, totp_backup_codes_hash, totp_backup_codes_remaining
		FROM users WHERE id = $1
	`, req.UserId).Scan(&encryptedSecret, &totpEnabled, pq.Array(&backupCodesHash), &backupCodesRemaining)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "database error")
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
		_, err = s.db.ExecContext(ctx, `
			UPDATE users 
			SET totp_backup_codes_hash = array_remove(totp_backup_codes_hash, $1),
			    totp_backup_codes_remaining = GREATEST(totp_backup_codes_remaining - 1, 0)
			WHERE id = $2
		`, backupCodesHash[idx], req.UserId)
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
		return nil, status.Error(codes.Internal, "failed to validate TOTP code")
	}

	s.log.Info("TOTP verified", zap.String("user_id", req.UserId), zap.Bool("valid", valid))
	return &pb.VerifyTOTPResponse{Valid: valid, BackupCodesRemaining: backupCodesRemaining}, nil
}

func (s *userServer) DisableTOTP(ctx context.Context, req *pb.DisableTOTPRequest) (*pb.DisableTOTPResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Passcode == "" {
		return nil, status.Error(codes.InvalidArgument, "passcode is required")
	}

	var encryptedSecret []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT totp_secret_encrypted FROM users 
		WHERE id = $1 AND totp_enabled = true
	`, req.UserId).Scan(&encryptedSecret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "TOTP not enabled for user")
		}
		return nil, status.Error(codes.Internal, "database error")
	}

	secret, err := s.totpService.DecryptSecret(encryptedSecret)
	if err != nil {
		s.log.Error("Failed to decrypt TOTP secret for disable", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to decrypt secret")
	}

	valid, err := s.totpService.ValidateTOTPCode(req.Passcode, secret)
	if err != nil {
		s.log.Warn("TOTP validation error during disable", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to validate TOTP code")
	}
	if !valid {
		return &pb.DisableTOTPResponse{Success: false, Message: "Invalid TOTP code"}, nil
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE users 
		SET totp_secret_encrypted = NULL, totp_enabled = false, 
		    totp_backup_codes_hash = NULL, totp_backup_codes_remaining = 0, updated_at = NOW()
		WHERE id = $1
	`, req.UserId)
	if err != nil {
		s.log.Error("Failed to disable TOTP", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to disable TOTP")
	}

	s.log.Info("TOTP disabled", zap.String("user_id", req.UserId))
	return &pb.DisableTOTPResponse{Success: true, Message: "TOTP disabled successfully"}, nil
}

func (s *userServer) ListHealthConditions(ctx context.Context, req *pb.ListHealthConditionsRequest) (*pb.ListHealthConditionsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, condition_type, condition_name, severity, diagnosed_at, is_active, notes, created_at, updated_at
		FROM user_health_conditions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, req.UserId)
	if err != nil {
		s.log.Error("Failed to query health conditions", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error("Failed to close rows for health conditions", zap.Error(closeErr))
		}
	}()

	conditions := make([]*pb.HealthCondition, 0)
	for rows.Next() {
		var c pb.HealthCondition
		var diagnosedAt sql.NullTime
		var notes sql.NullString
		if err := rows.Scan(&c.Id, &c.UserId, &c.ConditionType, &c.ConditionName, &c.Severity, &diagnosedAt, &c.IsActive, &notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			s.log.Error("Failed to scan health condition", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
		if diagnosedAt.Valid {
			c.DiagnosedAt = diagnosedAt.Time.Format("2006-01-02")
		}
		if notes.Valid {
			c.Notes = notes.String
		}
		conditions = append(conditions, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.ListHealthConditionsResponse{Conditions: conditions, Total: safeInt32(len(conditions))}, nil
}

func (s *userServer) UpsertHealthCondition(ctx context.Context, req *pb.UpsertHealthConditionRequest) (*pb.HealthCondition, error) {
	if req.UserId == "" || req.ConditionName == "" || req.ConditionType == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id, condition_type and condition_name are required")
	}
	var id string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO user_health_conditions (user_id, condition_type, condition_name, severity, diagnosed_at, is_active, notes, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id, condition_type, condition_name) DO UPDATE SET
			severity = EXCLUDED.severity,
			diagnosed_at = EXCLUDED.diagnosed_at,
			is_active = EXCLUDED.is_active,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id
	`, req.UserId, req.ConditionType, req.ConditionName, req.Severity, req.DiagnosedAt, req.IsActive, req.Notes).Scan(&id)
	if err != nil {
		s.log.Error("Failed to upsert health condition", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.HealthCondition{
		Id: id, UserId: req.UserId, ConditionType: req.ConditionType, ConditionName: req.ConditionName,
		Severity: req.Severity, DiagnosedAt: req.DiagnosedAt, IsActive: req.IsActive, Notes: req.Notes,
	}, nil
}

func (s *userServer) DeleteHealthCondition(ctx context.Context, req *pb.DeleteHealthConditionRequest) (*pb.DeleteHealthConditionResponse, error) {
	if req.UserId == "" || req.ConditionId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and condition_id are required")
	}
	if err := s.deleteRecord(ctx, "user_health_conditions", "id", req.UserId, req.ConditionId, "health condition"); err != nil {
		return nil, err
	}
	return &pb.DeleteHealthConditionResponse{Success: true, Message: "Health condition deleted"}, nil
}

func (s *userServer) ListBodyComposition(ctx context.Context, req *pb.ListBodyCompositionRequest) (*pb.ListBodyCompositionResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	limitClamped := limit
	if limitClamped > 10000 {
		limitClamped = 10000
	}

	var query string
	var args []interface{}

	switch {
	case req.From != "" && req.To != "":
		query = `SELECT id, user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source, created_at
			FROM user_body_composition
			WHERE user_id = $1 AND recorded_at >= $2 AND recorded_at <= $3
			ORDER BY recorded_at DESC LIMIT $4`
		args = []interface{}{req.UserId, req.From, req.To, limitClamped}
	case req.From != "":
		query = `SELECT id, user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source, created_at
			FROM user_body_composition
			WHERE user_id = $1 AND recorded_at >= $2
			ORDER BY recorded_at DESC LIMIT $3`
		args = []interface{}{req.UserId, req.From, limitClamped}
	case req.To != "":
		query = `SELECT id, user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source, created_at
			FROM user_body_composition
			WHERE user_id = $1 AND recorded_at <= $2
			ORDER BY recorded_at DESC LIMIT $3`
		args = []interface{}{req.UserId, req.To, limitClamped}
	default:
		query = `SELECT id, user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source, created_at
			FROM user_body_composition
			WHERE user_id = $1
			ORDER BY recorded_at DESC LIMIT $2`
		args = []interface{}{req.UserId, limitClamped}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		s.log.Error("Failed to query body composition", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error("Failed to close rows for body composition", zap.Error(closeErr))
		}
	}()

	records := make([]*pb.BodyCompositionRecord, 0)
	for rows.Next() {
		var r pb.BodyCompositionRecord
		if err := rows.Scan(&r.Id, &r.UserId, &r.RecordedAt, &r.WeightKg, &r.HeightCm, &r.Bmi, &r.BodyFatPercentage, &r.MuscleMassPercentage, &r.BoneMassPercentage, &r.WaterPercentage, &r.VisceralFatRating, &r.MetabolicAge, &r.Source, &r.CreatedAt); err != nil {
			s.log.Error("Failed to scan body composition record", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.ListBodyCompositionResponse{Records: records, Total: safeInt32(len(records))}, nil
}

func (s *userServer) CreateBodyComposition(ctx context.Context, req *pb.CreateBodyCompositionRequest) (*pb.BodyCompositionRecord, error) {
	if req.UserId == "" || req.WeightKg <= 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id and weight_kg are required")
	}
	var id string
	var recordedAt string
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO user_body_composition (user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source)
		VALUES ($1, COALESCE($2, NOW()), $3, $4, $5, $6, $7, $8, $9, $10, $11, COALESCE($12, 'manual'))
		RETURNING id, recorded_at
	`, req.UserId, req.RecordedAt, req.WeightKg, req.HeightCm, req.Bmi, req.BodyFatPercentage, req.MuscleMassPercentage, req.BoneMassPercentage, req.WaterPercentage, req.VisceralFatRating, req.MetabolicAge, req.Source).Scan(&id, &recordedAt)
	if err != nil {
		s.log.Error("Failed to create body composition record", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.BodyCompositionRecord{
		Id: id, UserId: req.UserId, RecordedAt: recordedAt, WeightKg: req.WeightKg, HeightCm: req.HeightCm,
		Bmi: req.Bmi, BodyFatPercentage: req.BodyFatPercentage, MuscleMassPercentage: req.MuscleMassPercentage,
		BoneMassPercentage: req.BoneMassPercentage, WaterPercentage: req.WaterPercentage,
		VisceralFatRating: req.VisceralFatRating, MetabolicAge: req.MetabolicAge, Source: req.Source,
	}, nil
}

func (s *userServer) ListMenstrualCycles(ctx context.Context, req *pb.ListMenstrualCyclesRequest) (*pb.ListMenstrualCyclesResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, cycle_start_date, cycle_end_date, flow_intensity, notes, created_at, updated_at
		FROM user_menstrual_cycles
		WHERE user_id = $1
		ORDER BY cycle_start_date DESC
	`, req.UserId)
	if err != nil {
		s.log.Error("Failed to query menstrual cycles", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error("Failed to close rows for health conditions", zap.Error(closeErr))
		}
	}()

	cycles := make([]*pb.MenstrualCycle, 0)
	for rows.Next() {
		var c pb.MenstrualCycle
		var cycleEndDate sql.NullTime
		var notes sql.NullString
		if err := rows.Scan(&c.Id, &c.UserId, &c.CycleStartDate, &cycleEndDate, &c.FlowIntensity, &notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			s.log.Error("Failed to scan menstrual cycle", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
		if cycleEndDate.Valid {
			c.CycleEndDate = cycleEndDate.Time.Format("2006-01-02")
		}
		if notes.Valid {
			c.Notes = notes.String
		}

		symptoms, err := s.queryCycleItems(ctx, "user_menstrual_symptoms", "symptom", c.Id)
		if err != nil {
			return nil, err
		}
		c.Symptoms = symptoms

		moods, err := s.queryCycleItems(ctx, "user_menstrual_moods", "mood", c.Id)
		if err != nil {
			return nil, err
		}
		c.Moods = moods

		cycles = append(cycles, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.ListMenstrualCyclesResponse{Cycles: cycles, Total: safeInt32(len(cycles))}, nil
}

func (s *userServer) queryCycleItems(ctx context.Context, table, column, cycleID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM %s WHERE cycle_id = $1", column, table), cycleID)
	if err != nil {
		s.log.Error("Failed to query cycle items", zap.Error(err), zap.String("table", table))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() { _ = rows.Close() }()

	items := make([]string, 0)
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err == nil {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		s.log.Error("Failed to iterate cycle item rows", zap.Error(err), zap.String("table", table))
		return nil, status.Error(codes.Internal, "database error")
	}
	return items, nil
}

func (s *userServer) CreateMenstrualCycle(ctx context.Context, req *pb.CreateMenstrualCycleRequest) (*pb.MenstrualCycle, error) {
	if req.UserId == "" || req.CycleStartDate == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and cycle_start_date are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			s.log.Error("Failed to rollback transaction", zap.Error(rollbackErr))
		}
	}()

	var cycleID string
	err = tx.QueryRowContext(ctx, `
		INSERT INTO user_menstrual_cycles (user_id, cycle_start_date, cycle_end_date, flow_intensity, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, req.UserId, req.CycleStartDate, nullIfEmpty(req.CycleEndDate), nullIfEmpty(req.FlowIntensity), req.Notes).Scan(&cycleID)
	if err != nil {
		s.log.Error("Failed to create menstrual cycle", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	for _, symptom := range req.Symptoms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_symptoms (cycle_id, symptom) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cycleID, symptom); err != nil {
			s.log.Error("Failed to insert symptom", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
	}
	for _, mood := range req.Moods {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_moods (cycle_id, mood) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cycleID, mood); err != nil {
			s.log.Error("Failed to insert mood", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.MenstrualCycle{
		Id: cycleID, UserId: req.UserId, CycleStartDate: req.CycleStartDate,
		CycleEndDate: req.CycleEndDate, FlowIntensity: req.FlowIntensity,
		Notes: req.Notes, Symptoms: req.Symptoms, Moods: req.Moods,
	}, nil
}

func (s *userServer) UpdateMenstrualCycle(ctx context.Context, req *pb.UpdateMenstrualCycleRequest) (*pb.MenstrualCycle, error) {
	if req.UserId == "" || req.CycleId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and cycle_id are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			s.log.Error("Failed to rollback transaction", zap.Error(rollbackErr))
		}
	}()

	_, err = tx.ExecContext(ctx, `
		UPDATE user_menstrual_cycles
		SET cycle_end_date = $1, flow_intensity = $2, notes = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
	`, nullIfEmpty(req.CycleEndDate), nullIfEmpty(req.FlowIntensity), req.Notes, req.CycleId, req.UserId)
	if err != nil {
		s.log.Error("Failed to update menstrual cycle", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	_, _ = tx.ExecContext(ctx, `DELETE FROM user_menstrual_symptoms WHERE cycle_id = $1`, req.CycleId)
	for _, symptom := range req.Symptoms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_symptoms (cycle_id, symptom) VALUES ($1, $2) ON CONFLICT DO NOTHING`, req.CycleId, symptom); err != nil {
			s.log.Error("Failed to update symptoms", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
	}
	_, _ = tx.ExecContext(ctx, `DELETE FROM user_menstrual_moods WHERE cycle_id = $1`, req.CycleId)
	for _, mood := range req.Moods {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_moods (cycle_id, mood) VALUES ($1, $2) ON CONFLICT DO NOTHING`, req.CycleId, mood); err != nil {
			s.log.Error("Failed to update moods", zap.Error(err))
			return nil, status.Error(codes.Internal, "database error")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Error(codes.Internal, "database error")
	}
	return &pb.MenstrualCycle{
		Id: req.CycleId, UserId: req.UserId, CycleStartDate: req.CycleStartDate,
		CycleEndDate: req.CycleEndDate, FlowIntensity: req.FlowIntensity,
		Notes: req.Notes, Symptoms: req.Symptoms, Moods: req.Moods,
	}, nil
}

func (s *userServer) DeleteMenstrualCycle(ctx context.Context, req *pb.DeleteMenstrualCycleRequest) (*pb.DeleteMenstrualCycleResponse, error) {
	if req.UserId == "" || req.CycleId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and cycle_id are required")
	}
	if err := s.deleteRecord(ctx, "user_menstrual_cycles", "id", req.UserId, req.CycleId, "menstrual cycle"); err != nil {
		return nil, err
	}
	return &pb.DeleteMenstrualCycleResponse{Success: true, Message: "Menstrual cycle deleted"}, nil
}

func (s *userServer) SyncFloData(ctx context.Context, req *pb.SyncFloDataRequest) (*pb.SyncFloDataResponse, error) {
	if req.UserId == "" || req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and access_token are required")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO device_provider_accounts (user_id, provider, provider_user_id, access_token, refresh_token, scopes, is_active)
		VALUES ($1, 'flo', $2, $3, $4, ARRAY['menstrual_cycle'], TRUE)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			is_active = TRUE,
			updated_at = NOW()
	`, req.UserId, req.UserId, req.AccessToken, nullIfStr(req.RefreshToken))

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store Flo tokens: %v", err)
	}

	return &pb.SyncFloDataResponse{
		Success:      true,
		Message:      "Flo tokens stored. Full sync pending API integration.",
		SyncedCycles: 0,
	}, nil
}

func (s *userServer) SyncOKOKData(ctx context.Context, req *pb.SyncOKOKDataRequest) (*pb.SyncOKOKDataResponse, error) {
	if req.UserId == "" || req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id and access_token are required")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO device_provider_accounts (user_id, provider, provider_user_id, access_token, refresh_token, scopes, is_active)
		VALUES ($1, 'okok', $2, $3, $4, ARRAY['body_composition'], TRUE)
		ON CONFLICT (user_id, provider)
		DO UPDATE SET
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			is_active = TRUE,
			updated_at = NOW()
	`, req.UserId, req.UserId, req.AccessToken, nullIfStr(req.RefreshToken))

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store OKOK tokens: %v", err)
	}

	return &pb.SyncOKOKDataResponse{
		Success:       true,
		Message:       "OKOK tokens stored. Full sync pending API integration.",
		SyncedRecords: 0,
	}, nil
}

func nullIfStr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func nullIfEmpty(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// ensurePgsodiumKey идемпотентно импортирует PII-ключ из DB_ENCRYPTION_KEY
// в keyring pgsodium (таблица pgsodium.key) и фиксирует его идентификатор
// в пакете db для использования в шифровании/расшифровке.
func (s *userServer) ensurePgsodiumKey(ctx context.Context) error {
	key := db.EncryptionKey()
	if len(key) == 0 {
		return errors.New("DB_ENCRYPTION_KEY not set; pgsodium keyring cannot be initialized")
	}

	var id int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM pgsodium.key WHERE name = $1`, db.PgsodiumKeyringName()).Scan(&id)
	if err == nil {
		db.SetPgsodiumKeyID(id)
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("query pgsodium key: %w", err)
	}

	hexKey := hex.EncodeToString(key)
	err = s.db.QueryRowContext(ctx,
		`SELECT pgsodium.import_key(CASE WHEN $1 ~ '^[0-9a-fA-F]{64}$' THEN decode($1, 'hex') ELSE convert_to($1, 'UTF8') END, $2)`,
		hexKey, db.PgsodiumKeyringName(),
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("import pgsodium key: %w", err)
	}
	db.SetPgsodiumKeyID(id)
	s.log.Info("Imported pgsodium PII key", zap.Int64("key_id", id))
	return nil
}

type pair struct {
	enc   string
	plain string
}

type piiTable struct {
	name  string
	idCol string
	pairs []pair
}

// reencryptPIIFromPgcrypto перекодирует существующие PII-поля,
// зашифрованные ранее через pgcrypto (pgp_sym_encrypt), в pgsodium (libsodium AEAD).
// Строки, уже зашифрованные через pgsodium, пропускаются.
func (s *userServer) reencryptPIIFromPgcrypto(ctx context.Context) {
	key := string(db.EncryptionKey())
	if key == "" {
		return
	}
	id := db.PgsodiumKeyID()
	if id == 0 {
		return
	}

	tables := []piiTable{
		{"users", "id", []pair{
			{"email_encrypted", "email"},
			{"full_name_encrypted", "full_name"},
			{"nickname_encrypted", "nickname"},
		}},
		{"email_verifications", "id", []pair{
			{"email_encrypted", "email"},
			{"token_encrypted", "token"},
		}},
	}

	for _, t := range tables {
		s.migrateTablePII(ctx, t, key, id)
	}
}

func (s *userServer) migrateTablePII(ctx context.Context, t piiTable, key string, id int64) {
	cols := []string{t.idCol}
	for _, p := range t.pairs {
		cols = append(cols, p.enc)
	}
	colList := strings.Join(cols, ", ")
	var selectBuilder strings.Builder
	selectBuilder.WriteString("SELECT ")
	selectBuilder.WriteString(colList)
	selectBuilder.WriteString(" FROM ")
	selectBuilder.WriteString(t.name)
	selectBuilder.WriteString(" WHERE ")
	selectBuilder.WriteString(t.pairs[0].enc)
	selectBuilder.WriteString(" IS NOT NULL")
	rows, err := s.db.QueryContext(ctx, selectBuilder.String())
	if err != nil {
		s.log.Error("Failed to scan PII rows for migration", zap.Error(err), zap.String("table", t.name))
		return
	}

	if err := rows.Err(); err != nil {
		s.log.Error("Failed to iterate PII rows for migration", zap.Error(err), zap.String("table", t.name))
		return
	}

	scanPtrs := make([]interface{}, len(cols))
	rowVals := make([]interface{}, len(cols))
	for i := range scanPtrs {
		scanPtrs[i] = &rowVals[i]
	}

	migrated := int64(0)
	for rows.Next() {
		if err := rows.Scan(scanPtrs...); err != nil {
			s.log.Error("Failed to scan PII row", zap.Error(err))
			continue
		}
		rowID := fmt.Sprint(rowVals[0])

		if s.migratePIIRow(ctx, t, key, id, rowID, rowVals) {
			migrated++
		}
	}
	if rowErr := rows.Err(); rowErr != nil {
		s.log.Error("Failed to iterate PII rows for migration", zap.Error(rowErr), zap.String("table", t.name))
		return
	}
	if closeErr := rows.Close(); closeErr != nil {
		s.log.Error("Failed to close rows during PII migration", zap.Error(closeErr), zap.String("table", t.name))
	}
	if migrated > 0 {
		s.log.Info("Re-encrypted PII from pgcrypto to pgsodium", zap.String("table", t.name), zap.Int64("rows", migrated))
	}
}

func (s *userServer) migratePIIRow(ctx context.Context, t piiTable, key string, id int64, rowID string, rowVals []interface{}) bool {
	var probe string
	if dErr := s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT convert_from(pgsodium.crypto_aead_det_decrypt($1, '', %d), 'UTF8')", id), rowVals[1],
	).Scan(&probe); dErr == nil {
		return false
	}

	setParts := make([]string, 0, len(t.pairs))
	args := make([]interface{}, 0, len(t.pairs)+1)
	ai := 1
	for i, p := range t.pairs {
		var plain sql.NullString
		if dErr := s.db.QueryRowContext(ctx, "SELECT pgp_sym_decrypt($1, $2)", rowVals[i+1], key).Scan(&plain); dErr != nil || !plain.Valid {
			s.log.Warn("Failed to pgcrypto-decrypt during PII migration",
				zap.Error(dErr), zap.String("table", t.name), zap.String("col", p.enc))
			return false
		}
		args = append(args, plain.String)
		setParts = append(setParts, fmt.Sprintf("%s = pgsodium.crypto_aead_det_encrypt($%d::text, '', %d)", p.enc, ai, id))
		ai++
	}
	if len(setParts) == 0 {
		return false
	}
	args = append(args, rowID)

	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE ")
	queryBuilder.WriteString(t.name)
	queryBuilder.WriteString(" SET ")
	queryBuilder.WriteString(strings.Join(setParts, ", "))
	queryBuilder.WriteString(" WHERE ")
	queryBuilder.WriteString(t.idCol)
	queryBuilder.WriteString(" = $")
	queryBuilder.WriteString(strconv.Itoa(ai))
	query := queryBuilder.String()

	if _, uErr := s.db.ExecContext(ctx, query, args...); uErr != nil {
		s.log.Error("Failed to re-encrypt PII row", zap.Error(uErr), zap.String("table", t.name), zap.String("id", rowID))
		return false
	}
	return true
}

func (s *userServer) backfillEncryptedPII(ctx context.Context) {
	key := string(db.EncryptionKey())
	if key == "" {
		s.log.Warn("DB_ENCRYPTION_KEY not set; skipping PII backfill")
		return
	}
	id := db.PgsodiumKeyID()
	if id == 0 {
		s.log.Warn("pgsodium key not initialized; skipping PII backfill")
		return
	}
	enc := func(col string) string {
		return fmt.Sprintf("pgsodium.crypto_aead_det_encrypt(%s::text, '', %d)", col, id)
	}

	var usersQuery strings.Builder
	usersQuery.WriteString("UPDATE users SET ")
	usersQuery.WriteString("email_encrypted = ")
	usersQuery.WriteString(enc("email"))
	usersQuery.WriteString(", email_hash = encode(digest(lower(email), 'sha256'), 'hex'), ")
	usersQuery.WriteString("full_name_encrypted = ")
	usersQuery.WriteString(enc("full_name"))
	usersQuery.WriteString(", full_name_hash = encode(digest(lower(full_name), 'sha256'), 'hex'), ")
	usersQuery.WriteString("nickname_encrypted = ")
	usersQuery.WriteString(enc("nickname"))
	usersQuery.WriteString(", nickname_hash = encode(digest(lower(nickname), 'sha256'), 'hex') ")
	usersQuery.WriteString(" WHERE email_encrypted IS NULL")
	res, err := s.db.ExecContext(ctx, usersQuery.String())
	if err != nil {
		s.log.Error("Failed to backfill PII in users", zap.Error(err))
	} else {
		rows, _ := res.RowsAffected()
		s.log.Info("PII backfill complete for users", zap.Int64("updated", rows))
	}

	var emailVerificationsQuery strings.Builder
	emailVerificationsQuery.WriteString("UPDATE email_verifications SET ")
	emailVerificationsQuery.WriteString("email_encrypted = ")
	emailVerificationsQuery.WriteString(enc("email"))
	emailVerificationsQuery.WriteString(", token_encrypted = ")
	emailVerificationsQuery.WriteString(enc("token"))
	emailVerificationsQuery.WriteString(" WHERE email_encrypted IS NULL")
	_, err = s.db.ExecContext(ctx, emailVerificationsQuery.String())
	if err != nil {
		s.log.Error("Failed to backfill PII in email_verifications", zap.Error(err))
	}
}

func buildUserServer(database *sql.DB, log *logger.Logger, jwtPrivateKeyPEM, baseURL, googleClientID string, emailSender *email.Sender, totpService *totp.Service) *userServer {
	return &userServer{
		db:               database,
		log:              log,
		jwtPrivateKeyPEM: jwtPrivateKeyPEM,
		emailSender:      emailSender,
		baseURL:          baseURL,
		googleClientID:   googleClientID,
		totpService:      totpService,
	}
}

func createGRPCServer() *grpc.Server {
	var s *grpc.Server
	if creds, err := grpctls.GetServerTLSCredentials(); err == nil && creds != nil {
		s = grpc.NewServer(
			grpc.Creds(creds),
			grpc.ChainUnaryInterceptor(
				middleware.CorrelationIDGRPC(),
			),
			telemetry.ServerHandlerOption(),
		)
	} else {
		s = grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				middleware.CorrelationIDGRPC(),
			),
			telemetry.ServerHandlerOption(),
		)
	}
	return s
}

func registerAndServe(s *grpc.Server, svc *userServer, lis net.Listener, port string, log *logger.Logger) {
	pb.RegisterUserServiceServer(s, svc)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("user.UserService", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(s)

	log.Info("User service starting", zap.String("port", port))
	if err := s.Serve(lis); err != nil {
		log.Fatal("Failed to serve", zap.Error(err))
	}
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
	dbCfg := db.Config{
		Host:     config.GetEnv("DB_HOST", "localhost"),
		Port:     config.GetEnv("DB_PORT", "5432"),
		User:     config.GetEnv("POSTGRES_USER"),
		Password: config.GetEnv("POSTGRES_PASSWORD"),
		DBName:   config.GetEnv("POSTGRES_DB", "fitness"),
		SSLMode:  config.GetEnv("DB_SSLMODE", "disable"),
	}

	database, err := db.NewConnection(dbCfg)
	if err != nil {
		log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer func() {
		if closeErr := database.Close(); closeErr != nil {
			log.Error("Failed to close database connection", zap.Error(closeErr))
		}
	}()

	jwtPrivateKeyPEM := config.GetEnv("JWT_PRIVATE_KEY_PEM")
	if jwtPrivateKeyPEM == "" {
		log.Fatal("JWT_PRIVATE_KEY_PEM environment variable is required")
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
	emailSender := email.NewSender(emailCfg)
	baseURL := config.GetEnv("BASE_URL", "https://localhost:8443")
	googleClientID := config.GetEnv("GOOGLE_CLIENT_ID")
	if googleClientID == "" {
		log.Fatal("GOOGLE_CLIENT_ID environment variable is required for Google OAuth")
	}

	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	svc := buildUserServer(database, log, jwtPrivateKeyPEM, baseURL, googleClientID, emailSender, totp.NewService(totpEncryptor))
	if err := svc.ensurePgsodiumKey(context.Background()); err != nil {
		log.Fatal("Failed to initialize pgsodium keyring", zap.Error(err))
	}
	svc.reencryptPIIFromPgcrypto(context.Background())
	svc.backfillEncryptedPII(context.Background())

	s := createGRPCServer()
	registerAndServe(s, svc, lis, port, log)
}
