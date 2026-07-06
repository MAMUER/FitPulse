// Package validator provides input validation utilities for API requests.
package validator

import (
	"errors"
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/user"
)

var (
	ErrEmailRequired    = errors.New("email is required")
	ErrInvalidEmail     = errors.New("invalid email format")
	ErrPasswordRequired = errors.New("password is required")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrFullNameRequired = errors.New("full name is required")
	ErrRoleRequired     = errors.New("role is required")
	ErrInvalidRole      = errors.New("invalid role, must be client or admin")
	ErrAgeOutOfRange    = errors.New("age must be between 0 and 150")
	ErrHeightOutOfRange = errors.New("height_cm must be between 50 and 300")
	ErrWeightOutOfRange = errors.New("weight_kg must be between 1 and 500")
	ErrInvalidFitness   = errors.New("fitness_level must be beginner, intermediate, or advanced")
	ErrInvalidGender    = errors.New("gender must be male, female, or other")
)

// emailRegex проверяет формат email
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NilRequestError возвращает стандартизированную ошибку для nil запроса
func NilRequestError() error {
	return status.Error(codes.InvalidArgument, "request is nil")
}

// ValidateRegisterRequest проверяет данные регистрации
func ValidateRegisterRequest(req *pb.RegisterRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.Email == "" {
		return status.Error(codes.InvalidArgument, ErrEmailRequired.Error())
	}
	if !emailRegex.MatchString(req.Email) {
		return status.Error(codes.InvalidArgument, ErrInvalidEmail.Error())
	}
	if req.Password == "" {
		return status.Error(codes.InvalidArgument, ErrPasswordRequired.Error())
	}
	if len(req.Password) < 8 {
		return status.Error(codes.InvalidArgument, ErrPasswordTooShort.Error())
	}
	if req.FullName == "" {
		return status.Error(codes.InvalidArgument, ErrFullNameRequired.Error())
	}
	if req.Role == "" {
		return status.Error(codes.InvalidArgument, ErrRoleRequired.Error())
	}
	validRoles := map[string]bool{"client": true, "admin": true}
	if !validRoles[req.Role] {
		return status.Error(codes.InvalidArgument, ErrInvalidRole.Error())
	}
	return nil
}

// ValidateLoginRequest проверяет данные для входа
func ValidateLoginRequest(req *pb.LoginRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.Email == "" {
		return status.Error(codes.InvalidArgument, ErrEmailRequired.Error())
	}
	if req.Password == "" {
		return status.Error(codes.InvalidArgument, ErrPasswordRequired.Error())
	}
	return nil
}

// ValidateProfileUpdate проверяет данные обновления профиля
func ValidateProfileUpdate(req *pb.UpdateProfileRequest) error {
	if req == nil {
		return NilRequestError()
	}
	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}
	if err := validateProfileFullName(req); err != nil {
		return err
	}
	if err := validateProfileAge(req); err != nil {
		return err
	}
	if err := validateProfileHeight(req); err != nil {
		return err
	}
	if err := validateProfileWeight(req); err != nil {
		return err
	}
	if err := validateProfileFitnessLevel(req); err != nil {
		return err
	}
	if err := validateProfileGender(req); err != nil {
		return err
	}
	return nil
}

func validateProfileFullName(req *pb.UpdateProfileRequest) error {
	if req.FullName == nil {
		return nil
	}
	fn := *req.FullName
	if len(fn) < 2 || len(fn) > 100 {
		return status.Error(codes.InvalidArgument, "nick must be 2-100 chars")
	}
	return nil
}

func validateProfileAge(req *pb.UpdateProfileRequest) error {
	if req.Age != nil && (*req.Age < 0 || *req.Age > 150) {
		return status.Error(codes.InvalidArgument, ErrAgeOutOfRange.Error())
	}
	return nil
}

func validateProfileHeight(req *pb.UpdateProfileRequest) error {
	if req.HeightCm != nil && (*req.HeightCm < 50 || *req.HeightCm > 300) {
		return status.Error(codes.InvalidArgument, ErrHeightOutOfRange.Error())
	}
	return nil
}

func validateProfileWeight(req *pb.UpdateProfileRequest) error {
	if req.WeightKg != nil && (*req.WeightKg < 1 || *req.WeightKg > 500) {
		return status.Error(codes.InvalidArgument, ErrWeightOutOfRange.Error())
	}
	return nil
}

func validateProfileFitnessLevel(req *pb.UpdateProfileRequest) error {
	validFitnessLevels := map[string]bool{"": true, "beginner": true, "intermediate": true, "advanced": true}
	if req.FitnessLevel != nil && !validFitnessLevels[*req.FitnessLevel] {
		return status.Error(codes.InvalidArgument, ErrInvalidFitness.Error())
	}
	return nil
}

func validateProfileGender(req *pb.UpdateProfileRequest) error {
	validGenders := map[string]bool{"": true, "male": true, "female": true, "other": true}
	if req.Gender != nil && !validGenders[*req.Gender] {
		return status.Error(codes.InvalidArgument, ErrInvalidGender.Error())
	}
	return nil
}
