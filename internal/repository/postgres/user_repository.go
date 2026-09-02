package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
	"github.com/MAMUER/project/internal/domain/port"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) port.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (id, email, email_hash, password_hash, full_name, full_name_hash, role, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Email, user.PasswordHash, user.FullName, user.FullName,
		user.Role, user.EmailVerified, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		return apperrors.Internal("failed to create user", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at
		FROM users WHERE id = $1
	`
	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Internal("failed to get user", err)
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at
		FROM users WHERE email = $1
	`
	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Internal("failed to get user by email", err)
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users SET full_name = $1, updated_at = $2 WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, user.FullName, time.Now(), user.ID)
	if err != nil {
		return apperrors.Internal("failed to update user", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Internal("failed to delete user", err)
	}
	return nil
}

func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]*entity.User, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, apperrors.Internal("failed to list users", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		user := &entity.User{}
		if err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
			&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan user", err)
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, apperrors.Internal("failed to count users", err)
	}
	return count, nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, apperrors.Internal("failed to check user existence", err)
	}
	return exists, nil
}

type Invite struct {
	Code      string
	Role      string
	Specialty string
	MaxUses   int
	UsedCount int
	IsActive  bool
	CreatedAt time.Time
	InviteURL string
}

type InviteRepository interface {
	Create(ctx context.Context, invite *Invite) error
	GetByCode(ctx context.Context, code string) (*Invite, error)
	List(ctx context.Context, page, pageSize int) ([]*Invite, int, error)
	Revoke(ctx context.Context, code string) error
}

type inviteRepository struct {
	db *sql.DB
}

func NewInviteRepository(db *sql.DB) InviteRepository {
	return &inviteRepository{db: db}
}

func (r *inviteRepository) Create(ctx context.Context, invite *Invite) error {
	query := `
		INSERT INTO invites (code, role, specialty, max_uses, used_count, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		invite.Code, invite.Role, invite.Specialty, invite.MaxUses, invite.UsedCount, invite.IsActive, invite.CreatedAt,
	)
	if err != nil {
		return apperrors.Internal("failed to create invite", err)
	}
	return nil
}

func (r *inviteRepository) GetByCode(ctx context.Context, code string) (*Invite, error) {
	query := `
		SELECT code, role, specialty, max_uses, used_count, is_active, created_at
		FROM invites WHERE code = $1
	`
	invite := &Invite{}
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&invite.Code, &invite.Role, &invite.Specialty, &invite.MaxUses, &invite.UsedCount, &invite.IsActive, &invite.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("invite not found")
		}
		return nil, apperrors.Internal("failed to get invite", err)
	}
	return invite, nil
}

func (r *inviteRepository) List(ctx context.Context, page, pageSize int) ([]*Invite, int, error) {
	offset := (page - 1) * pageSize
	query := `
		SELECT code, role, specialty, max_uses, used_count, is_active, created_at
		FROM invites ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list invites", err)
	}
	defer rows.Close()

	var invites []*Invite
	for rows.Next() {
		invite := &Invite{}
		if err := rows.Scan(
			&invite.Code, &invite.Role, &invite.Specialty, &invite.MaxUses, &invite.UsedCount, &invite.IsActive, &invite.CreatedAt,
		); err != nil {
			return nil, 0, apperrors.Internal("failed to scan invite", err)
		}
		invites = append(invites, invite)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM invites`).Scan(&total); err != nil {
		return nil, 0, apperrors.Internal("failed to count invites", err)
	}

	return invites, total, nil
}

func (r *inviteRepository) Revoke(ctx context.Context, code string) error {
	query := `UPDATE invites SET is_active = false WHERE code = $1`
	_, err := r.db.ExecContext(ctx, query, code)
	if err != nil {
		return apperrors.Internal("failed to revoke invite", err)
	}
	return nil
}

type ProfileRepository interface {
	GetProfile(ctx context.Context, userID string) (*entity.User, error)
	UpdateProfile(ctx context.Context, userID, fullName string, goals, contraindications []string, nutrition string, sleepHours float32) error
}

type profileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) GetProfile(ctx context.Context, userID string) (*entity.User, error) {
	return r.getUserByID(ctx, userID)
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

func (r *profileRepository) getUserByID(ctx context.Context, id string) (*entity.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, role, email_verified, created_at, updated_at
		FROM users WHERE id = $1
	`
	user := &entity.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName,
		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Internal("failed to get user", err)
	}
	return user, nil
}

type HealthCondition struct {
	ID            string
	UserID        string
	ConditionType string
	Description   string
	CreatedAt     time.Time
}

type HealthConditionRepository interface {
	Create(ctx context.Context, condition *HealthCondition) (*HealthCondition, error)
	List(ctx context.Context, userID, conditionType string) ([]*HealthCondition, error)
	Delete(ctx context.Context, id string) error
}

type healthConditionRepository struct {
	db *sql.DB
}

func NewHealthConditionRepository(db *sql.DB) HealthConditionRepository {
	return &healthConditionRepository{db: db}
}

func (r *healthConditionRepository) Create(ctx context.Context, condition *HealthCondition) (*HealthCondition, error) {
	query := `
		INSERT INTO health_conditions (id, user_id, condition_type, description, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		condition.ID, condition.UserID, condition.ConditionType, condition.Description, condition.CreatedAt,
	).Scan(&condition.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create health condition", err)
	}
	return condition, nil
}

func (r *healthConditionRepository) List(ctx context.Context, userID, conditionType string) ([]*HealthCondition, error) {
	query := `
		SELECT id, user_id, condition_type, description, created_at
		FROM health_conditions WHERE user_id = $1
	`
	args := []interface{}{userID}
	if conditionType != "" {
		query += " AND condition_type = $2"
		args = append(args, conditionType)
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal("failed to list health conditions", err)
	}
	defer rows.Close()

	var conditions []*HealthCondition
	for rows.Next() {
		condition := &HealthCondition{}
		if err := rows.Scan(
			&condition.ID, &condition.UserID, &condition.ConditionType, &condition.Description, &condition.CreatedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan health condition", err)
		}
		conditions = append(conditions, condition)
	}
	return conditions, nil
}

func (r *healthConditionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM health_conditions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Internal("failed to delete health condition", err)
	}
	return nil
}

type BodyCompositionRepository interface {
	Create(ctx context.Context, bc *entity.BodyComposition) (*entity.BodyComposition, error)
	List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*entity.BodyComposition, error)
}

type bodyCompositionRepository struct {
	db *sql.DB
}

func NewBodyCompositionRepository(db *sql.DB) BodyCompositionRepository {
	return &bodyCompositionRepository{db: db}
}

func (r *bodyCompositionRepository) Create(ctx context.Context, bc *entity.BodyComposition) (*entity.BodyComposition, error) {
	query := `
		INSERT INTO body_composition (id, user_id, weight_kg, height_cm, bmi, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		bc.ID, bc.UserID, bc.WeightKG, bc.HeightCM, bc.BMI, bc.RecordedAt,
	).Scan(&bc.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create body composition", err)
	}
	return bc, nil
}

func (r *bodyCompositionRepository) List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*entity.BodyComposition, error) {
	query := `
		SELECT id, user_id, weight_kg, height_cm, bmi, recorded_at
		FROM body_composition WHERE user_id = $1
	`
	args := []interface{}{userID}
	argCount := 1

	if from != nil {
		argCount++
		query += fmt.Sprintf(" AND recorded_at >= $%d", argCount)
		args = append(args, *from)
	}
	if to != nil {
		argCount++
		query += fmt.Sprintf(" AND recorded_at <= $%d", argCount)
		args = append(args, *to)
	}
	query += " ORDER BY recorded_at DESC LIMIT $" + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal("failed to list body composition", err)
	}
	defer rows.Close()

	var records []*entity.BodyComposition
	for rows.Next() {
		bc := &entity.BodyComposition{}
		if err := rows.Scan(
			&bc.ID, &bc.UserID, &bc.WeightKG, &bc.HeightCM, &bc.BMI, &bc.RecordedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan body composition", err)
		}
		records = append(records, bc)
	}
	return records, nil
}

type MenstrualCycleRepository interface {
	Create(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	List(ctx context.Context, userID string) ([]*entity.MenstrualCycle, error)
	Update(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error)
	Delete(ctx context.Context, id string) error
}

type menstrualCycleRepository struct {
	db *sql.DB
}

func NewMenstrualCycleRepository(db *sql.DB) MenstrualCycleRepository {
	return &menstrualCycleRepository{db: db}
}

func (r *menstrualCycleRepository) Create(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error) {
	query := `
		INSERT INTO menstrual_cycles (id, user_id, start_date, end_date, flow_level, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		cycle.ID, cycle.UserID, cycle.StartDate, cycle.EndDate, cycle.FlowLevel, cycle.Notes, cycle.CreatedAt,
	).Scan(&cycle.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create menstrual cycle", err)
	}
	return cycle, nil
}

func (r *menstrualCycleRepository) List(ctx context.Context, userID string) ([]*entity.MenstrualCycle, error) {
	query := `
		SELECT id, user_id, start_date, end_date, flow_level, notes, created_at
		FROM menstrual_cycles WHERE user_id = $1 ORDER BY start_date DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to list menstrual cycles", err)
	}
	defer rows.Close()

	var cycles []*entity.MenstrualCycle
	for rows.Next() {
		cycle := &entity.MenstrualCycle{}
		if err := rows.Scan(
			&cycle.ID, &cycle.UserID, &cycle.StartDate, &cycle.EndDate, &cycle.FlowLevel, &cycle.Notes, &cycle.CreatedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan menstrual cycle", err)
		}
		cycles = append(cycles, cycle)
	}
	return cycles, nil
}

func (r *menstrualCycleRepository) Update(ctx context.Context, cycle *entity.MenstrualCycle) (*entity.MenstrualCycle, error) {
	query := `
		UPDATE menstrual_cycles SET start_date = $1, end_date = $2, flow_level = $3, notes = $4
		WHERE id = $5
	`
	_, err := r.db.ExecContext(ctx, query, cycle.StartDate, cycle.EndDate, cycle.FlowLevel, cycle.Notes, cycle.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to update menstrual cycle", err)
	}
	return cycle, nil
}

func (r *menstrualCycleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM menstrual_cycles WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return apperrors.Internal("failed to delete menstrual cycle", err)
	}
	return nil
}

type AchievementRepository interface {
	Create(ctx context.Context, achievement *entity.Achievement) (*entity.Achievement, error)
	List(ctx context.Context, userID string) ([]*entity.Achievement, error)
}

type achievementRepository struct {
	db *sql.DB
}

func NewAchievementRepository(db *sql.DB) AchievementRepository {
	return &achievementRepository{db: db}
}

func (r *achievementRepository) Create(ctx context.Context, achievement *entity.Achievement) (*entity.Achievement, error) {
	query := `
		INSERT INTO achievements (id, user_id, type, title, description, earned_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	err := r.db.QueryRowContext(ctx, query,
		achievement.ID, achievement.UserID, achievement.Type, achievement.Title, achievement.Description, achievement.EarnedAt,
	).Scan(&achievement.ID)
	if err != nil {
		return nil, apperrors.Internal("failed to create achievement", err)
	}
	return achievement, nil
}

func (r *achievementRepository) List(ctx context.Context, userID string) ([]*entity.Achievement, error) {
	query := `
		SELECT id, user_id, type, title, description, earned_at
		FROM achievements WHERE user_id = $1 ORDER BY earned_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal("failed to list achievements", err)
	}
	defer rows.Close()

	var achievements []*entity.Achievement
	for rows.Next() {
		achievement := &entity.Achievement{}
		if err := rows.Scan(
			&achievement.ID, &achievement.UserID, &achievement.Type, &achievement.Title, &achievement.Description, &achievement.EarnedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan achievement", err)
		}
		achievements = append(achievements, achievement)
	}
	return achievements, nil
}
