// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type profileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) port.ProfileRepository {

	return &profileRepository{db: db}

}

func (r *profileRepository) queryUser(ctx context.Context, query string, args ...interface{}) (*entity.User, error) {

	user := &entity.User{}

	err := r.db.QueryRowContext(ctx, query, args...).Scan(

		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,

		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {

			return nil, apperrors.NotFound("user not found")

		}

		return nil, apperrors.Internal("failed to get user", err)

	}

	return user, nil

}

func (r *profileRepository) GetProfile(ctx context.Context, userID string) (*entity.User, error) {

	return r.queryUser(ctx, `

		SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at

		FROM users WHERE id = $1

	`, userID)

}

func (r *profileRepository) UpdateProfile(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error {

	query := `

		UPDATE users SET full_name = $1, updated_at = $2 WHERE id = $3

	`

	_, err := r.db.ExecContext(ctx, query, fullName, time.Now(), userID)

	if err != nil {

		return apperrors.Internal("failed to update profile", err)

	}

	return nil

}

func (r *profileRepository) UserExists(ctx context.Context, userID string) (bool, error) {

	var exists bool

	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)

	if err != nil {

		return false, apperrors.Internal("failed to check user existence", err)

	}

	return exists, nil

}

func (r *profileRepository) CreateProfile(ctx context.Context, userID string) error {

	_, err := r.db.ExecContext(ctx, `INSERT INTO user_profiles (user_id) VALUES ($1)`, userID)

	if err != nil {

		return apperrors.Internal("failed to create user profile", err)

	}

	return nil

}

func (r *profileRepository) UpsertProfile(ctx context.Context, userID string, data *port.ProfileData) error {

	query := `

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

	_, err := r.db.ExecContext(ctx, query,

		userID, data.Age, data.Gender, data.HeightCm, data.WeightKg, data.FitnessLevel, data.Nutrition, data.SleepHours,
	)

	if err != nil {

		return apperrors.Internal("failed to upsert user profile", err)

	}

	return nil

}
