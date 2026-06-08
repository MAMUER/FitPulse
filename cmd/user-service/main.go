package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	pb "github.com/MAMUER/project/api/gen/user"
	"github.com/MAMUER/project/internal/auth"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/email"
	"github.com/MAMUER/project/internal/logger"
	"github.com/MAMUER/project/internal/middleware"
	"github.com/MAMUER/project/internal/sanitize"
	"github.com/MAMUER/project/internal/validator"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
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
	db          *sql.DB
	log         *logger.Logger
	secret      string
	emailSender *email.Sender
	baseURL     string
}

func (s *userServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.log.Info("Register request", zap.String("email", req.Email))

	// Валидация входных данных
	if err := validator.ValidateRegisterRequest(req); err != nil {
		s.log.Warn("Invalid register request", zap.Error(err))
		return nil, err
	}

	// Санитизируем входные данные
	email := sanitize.String(req.Email)
	fullName := sanitize.String(req.FullName)

	// Проверка существования пользователя
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		s.log.Error("Database error checking user existence", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "user already exists")
	}

	// Хэширование пароля
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// Создание пользователя
	userID := uuid.New().String()
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO users (id, email, password_hash, full_name, role, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
    `, userID, email, string(hashed), fullName, req.Role)
	if err != nil {
		s.log.Error("Failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Генерация токена подтверждения email
	verificationToken := generateVerificationToken()
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO email_verifications (user_id, email, token, expires_at, used)
        VALUES ($1, $2, $3, NOW() + INTERVAL '24 hours', false)
    `, userID, email, verificationToken)
	if err != nil {
		s.log.Error("Failed to create email verification record", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create verification token")
	}

	// Отправка письма подтверждения (не блокирует регистрацию при ошибке)
	if s.emailSender != nil && s.baseURL != "" {
		if sendErr := s.emailSender.SendVerificationEmail(email, verificationToken, s.baseURL); sendErr != nil {
			s.log.Warn("Failed to send verification email (registration will proceed)",
				zap.Error(sendErr),
				zap.String("email", email))
		} else {
			s.log.Info("Verification email sent", zap.String("email", email))
		}
	}

	// Создание пустого профиля
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO user_profiles (user_id) VALUES ($1)
    `, userID)
	if err != nil {
		s.log.Warn("Failed to create user profile, user will need to complete profile manually",
			zap.Error(err),
			zap.String("user_id", userID))
	}

	return &pb.RegisterResponse{
		UserId:  userID,
		Message: "user created successfully. Verification token (dev only): " + verificationToken,
	}, nil
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
	err := s.db.QueryRowContext(ctx, `
        SELECT user_id, email, used, expires_at FROM email_verifications WHERE token = $1
    `, req.Token).Scan(&userID, &email, &used, &expiresAt)
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
		return nil, err
	}

	// Проверка подтверждения email
	var emailConfirmed bool
	err := s.db.QueryRowContext(ctx, "SELECT email_confirmed FROM users WHERE email = $1", req.Email).Scan(&emailConfirmed)
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
	err = s.db.QueryRowContext(ctx, `
        SELECT id, email, password_hash, role FROM users WHERE email = $1
    `, req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role)
	if errors.Is(err, sql.ErrNoRows) {
		// Возвращаем Unauthenticated вместо NotFound для безопасности
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if err != nil {
		s.log.Error("Database error during login", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}

	// Проверка пароля
	if cmpErr := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); cmpErr != nil {
		s.log.Info("Invalid login attempt", zap.String("email", req.Email))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Генерация JWT
	token, err := auth.GenerateJWT(user.ID, user.Email, user.Role, s.secret, 24)
	if err != nil {
		s.log.Error("Failed to generate JWT", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   24 * 3600,
		UserId:      user.ID,
		Role:        user.Role,
	}, nil
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

	err := s.db.QueryRowContext(ctx, `
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

func (s *userServer) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UserProfile, error) {
	// Валидация входных данных
	if err := validator.ValidateProfileUpdate(req); err != nil {
		s.log.Warn("Invalid profile update request", zap.Error(err))
		return nil, err
	}

	// Обновляем full_name и nickname в users table (если передан)
	if req.FullName != nil || req.Nickname != nil {
		_, err := s.db.ExecContext(ctx, `
			UPDATE users SET
				full_name = COALESCE($1, full_name),
				nickname = COALESCE($2, nickname),
				updated_at = NOW()
			WHERE id = $3
		`, req.FullName, req.Nickname, req.UserId)
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
	if len(req.Goals) > 0 {
		_, err = s.db.ExecContext(ctx, `DELETE FROM user_goals WHERE user_id = $1`, req.UserId)
		if err != nil {
			s.log.Error("Failed to delete old goals", zap.Error(err), zap.String("user_id", req.UserId))
			return nil, status.Error(codes.Internal, "failed to update goals")
		}
		for _, goal := range req.Goals {
			_, err = s.db.ExecContext(ctx,
				`INSERT INTO user_goals (user_id, goal) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				req.UserId, goal)
			if err != nil {
				s.log.Error("Failed to insert goal", zap.Error(err), zap.String("user_id", req.UserId))
				return nil, status.Error(codes.Internal, "failed to update goals")
			}
		}
	}

	// Обновляем contraindications
	if len(req.Contraindications) > 0 {
		_, err = s.db.ExecContext(ctx, `DELETE FROM user_contraindications WHERE user_id = $1`, req.UserId)
		if err != nil {
			s.log.Error("Failed to delete old contraindications", zap.Error(err), zap.String("user_id", req.UserId))
			return nil, status.Error(codes.Internal, "failed to update contraindications")
		}
		for _, c := range req.Contraindications {
			_, err = s.db.ExecContext(ctx,
				`INSERT INTO user_contraindications (user_id, contraindication) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				req.UserId, c)
			if err != nil {
				s.log.Error("Failed to insert contraindication", zap.Error(err), zap.String("user_id", req.UserId))
				return nil, status.Error(codes.Internal, "failed to update contraindications")
			}
		}
	}

	// Возвращаем обновленный профиль
	return s.GetProfile(ctx, &pb.GetProfileRequest{UserId: req.UserId})
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
	if err = bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(req.CurrentPassword)); err != nil {
		s.log.Warn("Password change failed: incorrect current password", zap.String("user_id", req.UserId))
		return nil, status.Error(codes.Unauthenticated, "current password is incorrect")
	}

	// Hash new password
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
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

	// Check if nickname is unique
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE nickname = $1 AND id != $2)", req.NewNickname, req.UserId).Scan(&exists)
	if err != nil {
		s.log.Error("Failed to check nickname uniqueness", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	if exists {
		return nil, status.Error(codes.AlreadyExists, "nickname already taken")
	}

	// Update nickname
	_, err = s.db.ExecContext(ctx, "UPDATE users SET nickname = $1, updated_at = NOW() WHERE id = $2", req.NewNickname, req.UserId)
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

	// In real implementation, trigger sync with wearable emulator or real API
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

// GetAchievements retrieves user's achievements (stub implementation).
func (s *userServer) GetAchievements(ctx context.Context, req *pb.GetAchievementsRequest) (*pb.GetAchievementsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	// In real implementation, query achievements from database
	// For now, return mock achievements
	achievements := []*pb.Achievement{
		{
			AchievementId: "first_workout",
			Title:         "First Workout",
			Description:   "Completed your first training session",
			EarnedDate:    "2024-01-15T10:00:00Z",
			IconUrl:       "/icons/first_workout.png",
		},
		{
			AchievementId: "week_streak",
			Title:         "Week Streak",
			Description:   "Worked out for 7 consecutive days",
			EarnedDate:    "2024-02-01T10:00:00Z",
			IconUrl:       "/icons/week_streak.png",
		},
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
	rows, err := s.db.QueryContext(ctx, `
        SELECT u.id, u.email, u.full_name, u.role, u.created_at, u.updated_at
        FROM users u
        WHERE ($1 = '' OR u.role = $1)
        ORDER BY u.created_at DESC
        LIMIT $2 OFFSET $3
    `, req.Role, req.PageSize, offset)
	if err != nil {
		s.log.Error("Failed to list users", zap.Error(err))
		return nil, status.Error(codes.Internal, "database error")
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Warn("Failed to close rows", zap.Error(closeErr))
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

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.GetPassword()), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Создаём пользователя
	userID := uuid.New().String()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, role, email_confirmed)
		VALUES ($1, $2, $3, $4, $5, true)
	`, userID, sanitize.String(req.GetEmail()), string(hashedPassword), sanitize.String(req.GetFullName()), finalRole)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, status.Error(codes.AlreadyExists, "email already exists")
		}
		s.log.Error("Failed to create user", zap.Error(err))
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Генерируем JWT (токен возвращается при login, не при регистрации)
	_, err = auth.GenerateJWT(userID, req.GetEmail(), finalRole, s.secret, 24)
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

func main() {
	log := logger.New("user-service")
	defer func() { _ = log.Sync() }()

	port := os.Getenv("USER_SERVICE_PORT")
	if port == "" {
		port = "50051"
	}

	dbCfg := db.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("POSTGRES_DB"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
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

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// SMTP email configuration
	emailCfg := email.LoadConfig()
	emailSender := email.NewSender(emailCfg)
	baseURL := getEnvOrDefault("BASE_URL", "https://localhost:8443")

	lis, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen", zap.Error(err))
	}

	s := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.CorrelationIDGRPC()),
	)
	pb.RegisterUserServiceServer(s, &userServer{
		db:          database,
		log:         log,
		secret:      secret,
		emailSender: emailSender,
		baseURL:     baseURL,
	})

	// Register gRPC health server
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("user.UserService", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(s)

	log.Info("User service starting", zap.String("port", port))
	if err := s.Serve(lis); err != nil {
		log.Fatal("Failed to serve", zap.Error(err))
	}
}
