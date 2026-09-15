// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/domain/entity"
)

// PgsodiumUserRepository handles user operations with pgsodium encryption.

type PgsodiumUserRepository struct {
	db *sql.DB
}

const (
	errFailedToGenerateNonce = "failed to generate nonce"

	errFailedToCreateUser = "failed to create user"

	errUserNotFound = "user not found"

	errFailedToDecryptEmail = "failed to decrypt email"

	errFailedToDecryptFullName = "failed to decrypt full name"

	errFailedToUpdateUser = "failed to update user"

	errFailedToUpdateNickname = "failed to update nickname"

	errFailedToDeleteUser = "failed to delete user"

	errFailedToListUsers = "failed to list users"

	errFailedToCountUsers = "failed to count users"

	errFailedToGetUser = "failed to get user"

	errFailedToScanUser = "failed to scan user"

	errFailedToIterateUsers = "failed to iterate users"

	errFailedToGetEmail = "failed to get email"

	errFailedToGetUserClaims = "failed to get user claims"

	errFailedToCheckExistence = "failed to check user existence"

	errFailedToCreateEmailVerification = "failed to create email verification record"

	errDatabaseCheckToken = "database error checking verification token"

	errDatabaseDuringLogin = "database error during login"

	errDatabaseGetProfile = "database error getting profile"

	errFailedToGetPasswordHash = "failed to get password hash"

	errFailedToCheckNickname = "failed to check nickname"

	errFailedToUpdateProfilePhoto = "failed to update profile photo"

	errFailedToGetTOTPState = "failed to get TOTP state"

	errFailedToEnableTOTP = "failed to enable TOTP"

	errFailedToDisableTOTP = "failed to disable TOTP"

	errFailedToRemoveBackupCode = "failed to remove backup code"

	errFailedToDeleteProfile = "failed to delete profile"

	sqlSelectPrefix = "SELECT "

	sqlUsersByIDPrefix = "SELECT id, email_hash, password_hash, full_name_hash, role, email_verified, created_at, updated_at FROM users WHERE id = $1"

	sqlUsersByEmailHashPrefix = "SELECT id, email_hash, password_hash, full_name_hash, role, email_verified, created_at, updated_at FROM users WHERE email_hash = $1"

	sqlFromUsersByID = " FROM users WHERE id = $1"

	sqlCommaNewline = ",\n               "

	sqlValuesPrefix = "VALUES ($1, "

	sqlComma4Prefix = ", $4, "

	sql56Prefix = "$5, $6, "

	sqlProfileSelectPrefix = "SELECT u.id, "

	colUEmailEncrypted = "u.email_encrypted"

	colUEmailNonce = "u.email_nonce"

	colUFullNameEncrypted = "u.full_name_encrypted"

	colUFullNameNonce = "u.full_name_nonce"
)

func NewPgsodiumUserRepository(db *sql.DB) *PgsodiumUserRepository {

	return &PgsodiumUserRepository{db: db}

}

func (r *PgsodiumUserRepository) Create(ctx context.Context, user *entity.User) error {

	emailNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	fullNameNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	emailHash := db.BlindIndex(user.Email)

	fullNameHash := db.BlindIndex(user.FullName)

	query, args := buildUserInsertQuery(userInsertData{

		userID: user.ID, email: user.Email, emailHash: emailHash, passwordHash: user.PasswordHash,

		fullName: user.FullName, emailNonce: emailNonce, fullNameNonce: fullNameNonce,

		fullNameHash: fullNameHash, role: user.Role, emailConfirmed: false,
	})

	if _, execErr := r.db.ExecContext(ctx, query, args...); execErr != nil {

		return apperrors.Internal(errFailedToCreateUser, execErr)

	}

	return nil

}

func (r *PgsodiumUserRepository) CreateWithInvite(ctx context.Context, userID, email, emailHash, passwordHash, fullName, role string) error {

	emailNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	fullNameNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	fullNameHash := db.BlindIndex(fullName)

	query, args := buildUserInsertQuery(userInsertData{

		userID: userID, email: email, emailHash: emailHash, passwordHash: passwordHash,

		fullName: fullName, emailNonce: emailNonce, fullNameNonce: fullNameNonce,

		fullNameHash: fullNameHash, role: role, emailConfirmed: true,
	})

	if _, execErr := r.db.ExecContext(ctx, query, args...); execErr != nil {

		var pqErr *pq.Error

		if errors.As(execErr, &pqErr) && pqErr.Code == "23505" {

			return apperrors.Conflict("email already exists")

		}

		return apperrors.Internal(errFailedToCreateUser, execErr)

	}

	return nil

}

func (r *PgsodiumUserRepository) GetByID(ctx context.Context, id string) (*entity.User, error) {

	var user entity.User

	var emailHash, fullNameHash string

	err := r.db.QueryRowContext(ctx, sqlUsersByIDPrefix, id).Scan(

		&user.ID, &emailHash, &user.PasswordHash, &fullNameHash,

		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, apperrors.Internal(errFailedToGetUser, err)

	}

	if db.PgsodiumKeyID() > 0 {

		if err := r.decryptUserFields(ctx, &user); err != nil {

			return nil, err

		}

	}

	return &user, nil

}

func (r *PgsodiumUserRepository) decryptUserFields(ctx context.Context, user *entity.User) error {

	var email, fullName string

	emailQuery := strings.Builder{}

	emailQuery.WriteString(sqlSelectPrefix)

	emailQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

	emailQuery.WriteString(sqlFromUsersByID)

	if err := r.db.QueryRowContext(ctx, emailQuery.String(), user.ID).Scan(&email); err != nil { // NOSONAR go:S2077

		return apperrors.Internal(errFailedToDecryptEmail, err)

	}

	fullNameQuery := strings.Builder{}

	fullNameQuery.WriteString(sqlSelectPrefix)

	fullNameQuery.WriteString(db.PgsodiumDecryptParam("full_name_encrypted", "full_name_nonce", "full_name"))

	fullNameQuery.WriteString(sqlFromUsersByID)

	if err := r.db.QueryRowContext(ctx, fullNameQuery.String(), user.ID).Scan(&fullName); err != nil { // NOSONAR go:S2077

		return apperrors.Internal(errFailedToDecryptFullName, err)

	}

	user.Email = email

	user.FullName = fullName

	return nil

}

func (r *PgsodiumUserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {

	emailHash := db.BlindIndex(email)

	var user entity.User

	var fullNameHash string

	err := r.db.QueryRowContext(ctx, sqlUsersByEmailHashPrefix, emailHash).Scan(

		&user.ID, &user.Email, &user.PasswordHash, &fullNameHash,

		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, apperrors.Internal(errFailedToGetUser+" by email", err)

	}

	if db.PgsodiumKeyID() > 0 {

		var emailVal string

		emailQuery := strings.Builder{}

		emailQuery.WriteString(sqlSelectPrefix)

		emailQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

		emailQuery.WriteString(sqlFromUsersByID)

		if err := r.db.QueryRowContext(ctx, emailQuery.String(), user.ID).Scan(&emailVal); err != nil { // NOSONAR go:S2077

			return nil, apperrors.Internal(errFailedToDecryptEmail, err)

		}

		user.Email = emailVal

	}

	return &user, nil

}

func (r *PgsodiumUserRepository) GetByEmailHash(ctx context.Context, emailHash string) (*entity.User, error) {

	var user entity.User

	var fullNameHash string

	err := r.db.QueryRowContext(ctx, sqlUsersByEmailHashPrefix, emailHash).Scan(

		&user.ID, &user.Email, &user.PasswordHash, &fullNameHash,

		&user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, apperrors.Internal(errFailedToGetUser+" by email hash", err)

	}

	if db.PgsodiumKeyID() > 0 {

		var emailVal string

		emailQuery := strings.Builder{}

		emailQuery.WriteString(sqlSelectPrefix)

		emailQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

		emailQuery.WriteString(sqlFromUsersByID)

		if err := r.db.QueryRowContext(ctx, emailQuery.String(), user.ID).Scan(&emailVal); err != nil { // NOSONAR go:S2077

			return nil, apperrors.Internal(errFailedToDecryptEmail, err)

		}

		user.Email = emailVal

	}

	return &user, nil

}

func (r *PgsodiumUserRepository) GetEmailByID(ctx context.Context, userID string) (string, bool, error) {

	var email string

	var totpEnabled bool

	if db.PgsodiumKeyID() > 0 {

		query := strings.Builder{}

		query.WriteString(sqlSelectPrefix)

		query.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

		query.WriteString(", totp_enabled FROM users WHERE id = $1")

		if err := r.db.QueryRowContext(ctx, query.String(), userID).Scan(&email, &totpEnabled); err != nil { // NOSONAR go:S2077

			if errors.Is(err, sql.ErrNoRows) {

				return "", false, apperrors.NotFound(errUserNotFound)

			}

			return "", false, apperrors.Internal(errFailedToDecryptEmail, err)

		}

	} else {

		if err := r.db.QueryRowContext(ctx, "SELECT email, totp_enabled FROM users WHERE id = $1", userID).Scan(&email, &totpEnabled); err != nil {

			if errors.Is(err, sql.ErrNoRows) {

				return "", false, apperrors.NotFound(errUserNotFound)

			}

			return "", false, apperrors.Internal(errFailedToGetEmail, err)

		}

	}

	return email, totpEnabled, nil

}

func (r *PgsodiumUserRepository) GetClaimsByID(ctx context.Context, userID string) (email, role string, totpEnabled bool, backupCodesRemaining int32, err error) {

	if db.PgsodiumKeyID() > 0 {

		query := strings.Builder{}

		query.WriteString(sqlSelectPrefix)

		query.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

		query.WriteString(", role, totp_enabled, COALESCE(totp_backup_codes_remaining, 0) FROM users WHERE id = $1")

		err = r.db.QueryRowContext(ctx, query.String(), userID).Scan(&email, &role, &totpEnabled, &backupCodesRemaining) // NOSONAR go:S2077

	} else {

		err = r.db.QueryRowContext(ctx, "SELECT email, role, totp_enabled, COALESCE(totp_backup_codes_remaining, 0) FROM users WHERE id = $1", userID).Scan(&email, &role, &totpEnabled, &backupCodesRemaining)

	}

	if errors.Is(err, sql.ErrNoRows) {

		return "", "", false, 0, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return "", "", false, 0, apperrors.Internal(errFailedToGetUserClaims, err)

	}

	return email, role, totpEnabled, backupCodesRemaining, nil

}

func (r *PgsodiumUserRepository) GetProfileWithPgsodium(ctx context.Context, userID string) (*entity.User, error) {

	return r.loadProfile(ctx, userID)

}

func (r *PgsodiumUserRepository) loadProfile(ctx context.Context, userID string) (*entity.User, error) {

	var profile entity.User

	var nickname, profilePhotoURL sql.NullString

	var age sql.NullInt32

	var gender sql.NullString

	var heightCm sql.NullInt32

	var weightKg sql.NullFloat64

	var fitnessLevel sql.NullString

	var goals []string

	var nutrition sql.NullString

	var sleepHours sql.NullFloat64

	err := r.db.QueryRowContext(ctx, r.profileSelectQuery(), userID).Scan( // NOSONAR go:S2077

		&profile.ID, &profile.Email, &profile.FullName, &nickname, &profilePhotoURL, &profile.Role,

		&age, &gender, &heightCm, &weightKg, &fitnessLevel,

		&goals, &nutrition, &sleepHours,

		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, apperrors.Internal(errDatabaseGetProfile, err)

	}

	nullableProfileFields{
		Nickname:        nickname,
		ProfilePhotoURL: profilePhotoURL,
		Age:             age,
		Gender:          gender,
		HeightCm:        heightCm,
		WeightKg:        weightKg,
		FitnessLevel:    fitnessLevel,
		Nutrition:       nutrition,
		SleepHours:      sleepHours,
	}.applyTo(&profile)

	profile.Goals = goals

	return &profile, nil

}

func (r *PgsodiumUserRepository) profileSelectQuery() string {

	if db.PgsodiumKeyID() > 0 {

		var getProfileQuery strings.Builder

		getProfileQuery.WriteString(sqlProfileSelectPrefix)

		getProfileQuery.WriteString(db.PgsodiumDecryptParam(colUEmailEncrypted, colUEmailNonce, "email"))

		getProfileQuery.WriteString(sqlCommaNewline)

		getProfileQuery.WriteString(db.PgsodiumDecryptParam(colUFullNameEncrypted, colUFullNameNonce, "full_name"))

		getProfileQuery.WriteString(sqlCommaNewline)

		getProfileQuery.WriteString(db.PgsodiumDecryptParam("u.nickname_encrypted", "u.nickname_nonce", "nickname"))

		getProfileQuery.WriteString(",\n               u.profile_photo_url, u.role,\n               p.age, p.gender, p.height_cm, p.weight_kg, p.fitness_level,\n               p.goals, p.nutrition, p.sleep_hours,\n               u.created_at, u.updated_at\n            FROM users u\n            LEFT JOIN user_profiles_with_goals p ON u.id = p.user_id\n            WHERE u.id = $1")

		return getProfileQuery.String()

	}

	return `

		SELECT u.id, u.email, u.full_name, u.nickname, u.profile_photo_url, u.role,

		       p.age, p.gender, p.height_cm, p.weight_kg, p.fitness_level,

		       p.goals, p.nutrition, p.sleep_hours,

		       u.created_at, u.updated_at

		FROM users u

		LEFT JOIN user_profiles_with_goals p ON u.id = p.user_id

		WHERE u.id = $1

	`

}

type nullableProfileFields struct {
	Nickname        sql.NullString
	ProfilePhotoURL sql.NullString
	Age             sql.NullInt32
	Gender          sql.NullString
	HeightCm        sql.NullInt32
	WeightKg        sql.NullFloat64
	FitnessLevel    sql.NullString
	Nutrition       sql.NullString
	SleepHours      sql.NullFloat64
}

func (n nullableProfileFields) applyTo(profile *entity.User) {
	if n.Nickname.Valid {
		profile.Nickname = n.Nickname.String
	}
	if n.ProfilePhotoURL.Valid {
		profile.ProfilePhotoURL = n.ProfilePhotoURL.String
	}
	if n.Age.Valid {
		profile.Age = n.Age.Int32
	}
	if n.Gender.Valid {
		profile.Gender = n.Gender.String
	}
	if n.HeightCm.Valid {
		profile.HeightCm = n.HeightCm.Int32
	}
	if n.WeightKg.Valid {
		profile.WeightKg = n.WeightKg.Float64
	}
	if n.FitnessLevel.Valid {
		profile.FitnessLevel = n.FitnessLevel.String
	}
	if n.Nutrition.Valid {
		profile.Nutrition = n.Nutrition.String
	}
	if n.SleepHours.Valid {
		profile.SleepHours = float32(n.SleepHours.Float64)
	}
}

func (r *PgsodiumUserRepository) Update(ctx context.Context, user *entity.User) error {

	fullNameNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	fullNameHash := db.BlindIndex(user.FullName)

	query := strings.Builder{}

	query.WriteString("UPDATE users SET full_name_encrypted = ")

	query.WriteString(db.PgsodiumRandomEncryptParam(1, 2))

	query.WriteString(", full_name_nonce = $2, full_name_hash = $3, updated_at = $4 WHERE id = $5")

	_, execErr := r.db.ExecContext(ctx, query.String(), user.FullName, fullNameNonce, fullNameHash, time.Now(), user.ID) // NOSONAR go:S2077

	if execErr != nil {

		return apperrors.Internal(errFailedToUpdateUser, execErr)

	}

	return nil

}

func (r *PgsodiumUserRepository) UpdateUserDetails(ctx context.Context, userID, fullName, nickname string) error {

	fullNameNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	fullNameHash := db.BlindIndex(fullName)

	nicknameNonce, err := db.GenerateNonce()

	if err != nil {

		return apperrors.Internal(errFailedToGenerateNonce, err)

	}

	nicknameHash := db.BlindIndex(nickname)

	query := strings.Builder{}

	query.WriteString("UPDATE users SET\n\t\t\t\tfull_name_encrypted = CASE WHEN $1 IS NULL THEN full_name_encrypted ELSE ")

	query.WriteString(db.PgsodiumRandomEncryptParam(1, 2))

	query.WriteString(" END,\n\t\t\t\tfull_name_nonce = CASE WHEN $1 IS NULL THEN full_name_nonce ELSE $2 END,\n\t\t\t\tfull_name_hash = CASE WHEN $1 IS NULL THEN full_name_hash ELSE $3 END,\n\t\t\t\tnickname_encrypted = CASE WHEN $4 IS NULL THEN nickname_encrypted ELSE ")

	query.WriteString(db.PgsodiumRandomEncryptParam(4, 5))

	query.WriteString(" END,\n\t\t\t\tnickname_nonce = CASE WHEN $4 IS NULL THEN nickname_nonce ELSE $5 END,\n\t\t\t\tnickname_hash = CASE WHEN $4 IS NULL THEN nickname_hash ELSE $6 END,\n\t\t\t\tupdated_at = NOW()\n\t\t\tWHERE id = $7")

	_, execErr := r.db.ExecContext(ctx, query.String(), fullName, fullNameNonce, fullNameHash, nickname, nicknameNonce, nicknameHash, userID) // NOSONAR go:S2077

	if execErr != nil {

		return apperrors.Internal(errFailedToUpdateUser+" details", execErr)

	}

	return nil

}

func (r *PgsodiumUserRepository) UpdateNicknameWithPgsodium(ctx context.Context, userID, nickname string, nicknameNonce []byte, nicknameHash string) error {

	query := strings.Builder{}

	query.WriteString("UPDATE users SET nickname_encrypted = ")

	query.WriteString(db.PgsodiumRandomEncryptParam(1, 2))

	query.WriteString(", nickname_nonce = $2, nickname_hash = $3, updated_at = NOW() WHERE id = $4")

	_, execErr := r.db.ExecContext(ctx, query.String(), nickname, nicknameNonce, nicknameHash, userID) // NOSONAR go:S2077

	if execErr != nil {

		return apperrors.Internal(errFailedToUpdateNickname, execErr)

	}

	return nil

}

func (r *PgsodiumUserRepository) Delete(ctx context.Context, id string) error {

	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)

	if err != nil {

		return apperrors.Internal(errFailedToDeleteUser, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) List(ctx context.Context, page, pageSize int) ([]*entity.User, error) {

	offset := (page - 1) * pageSize

	var listUsersQuery strings.Builder

	listUsersQuery.WriteString(sqlProfileSelectPrefix)

	listUsersQuery.WriteString(db.PgsodiumDecryptParam(colUEmailEncrypted, colUEmailNonce, "email"))

	listUsersQuery.WriteString(sqlCommaNewline)

	listUsersQuery.WriteString(db.PgsodiumDecryptParam(colUFullNameEncrypted, colUFullNameNonce, "full_name"))

	listUsersQuery.WriteString(",\n               u.role, u.email_verified, u.created_at, u.updated_at\n            FROM users u\n            WHERE 1=1")

	args := []interface{}{}

	argCount := 0

	argCount++

	fmt.Fprintf(&listUsersQuery, " ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listUsersQuery.String(), args...) // NOSONAR go:S2077

	if err != nil {

		return nil, apperrors.Internal(errFailedToListUsers, err)

	}

	defer func() { _ = rows.Close() }()

	var users []*entity.User

	for rows.Next() {

		user := &entity.User{}

		if err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt); err != nil {

			return nil, apperrors.Internal(errFailedToScanUser, err)

		}

		users = append(users, user)

	}

	if err := rows.Err(); err != nil {

		return nil, apperrors.Internal(errFailedToIterateUsers, err)

	}

	return users, nil

}

func (r *PgsodiumUserRepository) ListByRole(ctx context.Context, role string, page, pageSize int) ([]*entity.User, int, error) {

	offset := (page - 1) * pageSize

	var listUsersQuery strings.Builder

	listUsersQuery.WriteString(sqlProfileSelectPrefix)

	listUsersQuery.WriteString(db.PgsodiumDecryptParam(colUEmailEncrypted, colUEmailNonce, "email"))

	listUsersQuery.WriteString(sqlCommaNewline)

	listUsersQuery.WriteString(db.PgsodiumDecryptParam(colUFullNameEncrypted, colUFullNameNonce, "full_name"))

	listUsersQuery.WriteString(",\n               u.role, u.email_verified, u.created_at, u.updated_at\n            FROM users u\n            WHERE u.role = $1")

	args := []interface{}{role}

	argCount := 1

	argCount++

	fmt.Fprintf(&listUsersQuery, " ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", argCount, argCount+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listUsersQuery.String(), args...) // NOSONAR go:S2077

	if err != nil {

		return nil, 0, apperrors.Internal(errFailedToListUsers, err)

	}

	defer func() { _ = rows.Close() }()

	var users []*entity.User

	for rows.Next() {

		user := &entity.User{}

		if err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt); err != nil {

			return nil, 0, apperrors.Internal(errFailedToScanUser, err)

		}

		users = append(users, user)

	}

	if err := rows.Err(); err != nil {

		return nil, 0, apperrors.Internal(errFailedToIterateUsers, err)

	}

	var total int

	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE role = $1", role).Scan(&total); err != nil {

		return nil, 0, apperrors.Internal(errFailedToCountUsers, err)

	}

	return users, total, nil

}

func (r *PgsodiumUserRepository) Count(ctx context.Context) (int, error) {

	var count int

	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)

	if err != nil {

		return 0, apperrors.Internal(errFailedToCountUsers, err)

	}

	return count, nil

}

func (r *PgsodiumUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {

	emailHash := db.BlindIndex(email)

	var exists bool

	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email_hash = $1)", emailHash).Scan(&exists)

	if err != nil {

		return false, apperrors.Internal(errFailedToCheckExistence, err)

	}

	return exists, nil

}

func (r *PgsodiumUserRepository) ConfirmEmail(ctx context.Context, token string) error {

	_, _, err := r.ConfirmEmailWithPgsodium(ctx, token)

	return err

}

func (r *PgsodiumUserRepository) ConfirmEmailWithPgsodium(ctx context.Context, token string) (userID, email string, err error) {

	var used bool

	var expiresAt sql.NullTime

	confirmEmailQuery := strings.Builder{}

	confirmEmailQuery.WriteString("SELECT user_id, ")

	confirmEmailQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

	confirmEmailQuery.WriteString(", used, expires_at FROM email_verifications WHERE token = $1")

	err = r.db.QueryRowContext(ctx, confirmEmailQuery.String(), token).Scan(&userID, &email, &used, &expiresAt) // NOSONAR go:S2077

	if errors.Is(err, sql.ErrNoRows) {

		return "", "", apperrors.InvalidArgument("invalid verification token")

	}

	if err != nil {

		return "", "", apperrors.Internal(errDatabaseCheckToken, err)

	}

	if used {

		return "", "", apperrors.InvalidArgument("verification token has already been used")

	}

	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {

		return "", "", apperrors.InvalidArgument("verification token has expired")

	}

	return userID, email, nil

}

func (r *PgsodiumUserRepository) CreateEmailVerification(ctx context.Context, userID, email, emailHash, verificationToken string, emailNonce, tokenNonce []byte) error {

	verificationQuery, verificationArgs := buildEmailVerificationInsertQuery(userID, email, emailHash, verificationToken, emailNonce, tokenNonce)

	if _, execErr := r.db.ExecContext(ctx, verificationQuery, verificationArgs...); execErr != nil {

		return apperrors.Internal(errFailedToCreateEmailVerification, execErr)

	}

	return nil

}

func (r *PgsodiumUserRepository) LoginWithPgsodium(ctx context.Context, email string) (*entity.User, error) {

	emailHash := db.BlindIndex(email)

	var emailConfirmed bool

	err := r.db.QueryRowContext(ctx, "SELECT email_confirmed FROM users WHERE email_hash = $1", emailHash).Scan(&emailConfirmed)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, apperrors.Internal("database error checking email confirmation", err)

	}

	if !emailConfirmed {

		return nil, apperrors.Unauthorized("email not confirmed")

	}

	var user entity.User

	loginQuery := strings.Builder{}

	loginQuery.WriteString("SELECT id, ")

	loginQuery.WriteString(db.PgsodiumDecryptParam("email_encrypted", "email_nonce", "email"))

	loginQuery.WriteString(", password_hash, role FROM users WHERE email_hash = $1")

	err = r.db.QueryRowContext(ctx, loginQuery.String(), emailHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role) // NOSONAR go:S2077

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.Unauthorized("invalid credentials")

	}

	if err != nil {

		return nil, apperrors.Internal(errDatabaseDuringLogin, err)

	}

	user.EmailVerified = emailConfirmed

	return &user, nil

}

func (r *PgsodiumUserRepository) CreateGoogleUser(ctx context.Context, googleSub, emailHash, emailVal, nickname, nicknameHash string) (userID, role string, emailConfirmed bool, err error) {

	nicknameNonce, err := db.GenerateNonce()

	if err != nil {

		return "", "", false, apperrors.Internal(errFailedToGenerateNonce, err)

	}

	emailNonce, err := db.GenerateNonce()

	if err != nil {

		return "", "", false, apperrors.Internal(errFailedToGenerateNonce, err)

	}

	userID = hex.EncodeToString(fmt.Appendf(nil, "%d", time.Now().UnixNano()))

	query, args := buildGoogleUserInsertQuery(googleUserInsertData{

		userID: userID, emailVal: emailVal, emailHash: emailHash, emailNonce: emailNonce,

		nickname: nickname, nicknameNonce: nicknameNonce, nicknameHash: nicknameHash, googleSub: googleSub,
	})

	_, insertErr := r.db.ExecContext(ctx, query, args...)

	if insertErr != nil {

		var pqErr *pq.Error

		if errors.As(insertErr, &pqErr) && pqErr.Code == "23505" {

			return "", "", false, apperrors.Conflict("user already exists")

		}

		return "", "", false, apperrors.Internal(errFailedToCreateUser, insertErr)

	}

	role = "client"

	emailConfirmed = true

	return userID, role, emailConfirmed, nil

}

type userInsertData struct {
	userID, email, emailHash, passwordHash, fullName string

	emailNonce, fullNameNonce []byte

	fullNameHash, role string

	emailConfirmed bool
}

func buildUserInsertQuery(p userInsertData) (string, []interface{}) {

	var b strings.Builder

	b.WriteString("INSERT INTO users (id, email_encrypted, email_nonce, email_hash, password_hash, full_name_encrypted, full_name_nonce, full_name_hash, role, email_confirmed, created_at, updated_at) ")

	b.WriteString(sqlValuesPrefix)

	b.WriteString(db.PgsodiumRandomEncryptParam(2, 3))

	b.WriteString(sqlComma4Prefix)

	b.WriteString(sql56Prefix)

	b.WriteString(db.PgsodiumRandomEncryptParam(7, 8))

	b.WriteString(", $9, ")

	b.WriteString("$10, ")

	b.WriteString("$11, NOW(), NOW())")

	args := []interface{}{p.userID, p.email, p.emailNonce, p.emailNonce, p.emailHash, p.passwordHash, p.fullName, p.fullNameNonce, p.fullNameNonce, p.fullNameHash, p.role, p.emailConfirmed}

	return b.String(), args

}

func buildEmailVerificationInsertQuery(userID, email, emailHash, verificationToken string, emailNonce, tokenNonce []byte) (string, []interface{}) {

	var b strings.Builder

	b.WriteString("INSERT INTO email_verifications (user_id, email_encrypted, email_nonce, email_hash, token, token_encrypted, token_nonce, expires_at, used) ")

	b.WriteString(sqlValuesPrefix)

	b.WriteString(db.PgsodiumRandomEncryptParam(2, 3))

	b.WriteString(sqlComma4Prefix)

	b.WriteString(sql56Prefix)

	b.WriteString(db.PgsodiumRandomEncryptParam(7, 8))

	b.WriteString(", $9, NOW() + INTERVAL '24 hours', false)")

	args := []interface{}{userID, email, emailNonce, emailNonce, emailHash, verificationToken, tokenNonce, tokenNonce}

	return b.String(), args

}

type googleUserInsertData struct {
	userID, emailVal, emailHash, nickname, googleSub string

	emailNonce, nicknameNonce []byte

	nicknameHash string
}

func buildGoogleUserInsertQuery(p googleUserInsertData) (string, []interface{}) {

	var b strings.Builder

	b.WriteString("INSERT INTO users (id, email_encrypted, email_nonce, email_hash, password_hash, nickname_encrypted, nickname_nonce, nickname_hash, role, provider, external_id, email_confirmed, created_at, updated_at) ")

	b.WriteString(sqlValuesPrefix)

	b.WriteString(db.PgsodiumRandomEncryptParam(2, 3))

	b.WriteString(sqlComma4Prefix)

	b.WriteString(sql56Prefix)

	b.WriteString(db.PgsodiumRandomEncryptParam(7, 8))

	b.WriteString(", $9, ")

	b.WriteString("$10, ")

	b.WriteString("'client', 'google', $11, true, NOW(), NOW())")

	args := []interface{}{p.userID, p.emailVal, p.emailNonce, p.emailNonce, p.emailHash, "", p.nickname, p.nicknameNonce, p.nicknameNonce, p.nicknameHash, p.googleSub}

	return b.String(), args

}

func (r *PgsodiumUserRepository) GetPasswordHash(ctx context.Context, userID string) (string, error) {

	var passwordHash string

	err := r.db.QueryRowContext(ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&passwordHash)

	if errors.Is(err, sql.ErrNoRows) {

		return "", apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return "", apperrors.Internal(errFailedToGetPasswordHash, err)

	}

	return passwordHash, nil

}

func (r *PgsodiumUserRepository) NicknameExists(ctx context.Context, nickname string, excludeUserID string) (bool, error) {

	var exists bool

	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE nickname = $1 AND id != $2)", nickname, excludeUserID).Scan(&exists)

	if err != nil {

		return false, apperrors.Internal(errFailedToCheckNickname, err)

	}

	return exists, nil

}

func (r *PgsodiumUserRepository) FindGoogleUser(ctx context.Context, googleSub string) (*entity.User, error) {

	var user entity.User

	var emailHash, fullNameHash string

	err := r.db.QueryRowContext(ctx, "SELECT id, email_hash, password_hash, full_name_hash, role, email_confirmed, created_at, updated_at FROM users WHERE provider = 'google' AND external_id = $1", googleSub).Scan(

		&user.ID, &emailHash, &user.PasswordHash, &fullNameHash, &user.Role, &user.EmailVerified, &user.CreatedAt, &user.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, apperrors.Internal("failed to find Google user", err)

	}

	if db.PgsodiumKeyID() > 0 {

		if err := r.decryptUserFields(ctx, &user); err != nil {

			return nil, err

		}

	}

	return &user, nil

}

func (r *PgsodiumUserRepository) LinkGoogleAccount(ctx context.Context, googleSub, userID string) error {

	_, err := r.db.ExecContext(ctx, `

		UPDATE users SET provider = 'google', external_id = $1, email_confirmed = true, updated_at = NOW()

		WHERE id = $2

	`, googleSub, userID)

	if err != nil {

		return apperrors.Internal("failed to link Google account", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) MarkEmailVerified(ctx context.Context, userID string) error {

	_, err := r.db.ExecContext(ctx, "UPDATE users SET email_verified = true, updated_at = NOW() WHERE id = $1", userID)

	if err != nil {

		return apperrors.Internal("failed to mark email as verified", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) CreateUserProfile(ctx context.Context, userID string) error {

	_, err := r.db.ExecContext(ctx, `INSERT INTO user_profiles (user_id) VALUES ($1)`, userID)

	if err != nil {

		return apperrors.Internal("failed to create user profile", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) GetUserRole(ctx context.Context, userID string) (string, error) {

	var role string

	err := r.db.QueryRowContext(ctx, "SELECT role FROM users WHERE id = $1", userID).Scan(&role)

	if errors.Is(err, sql.ErrNoRows) {

		return "", apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return "", apperrors.Internal("failed to get user role", err)

	}

	return role, nil

}

func (r *PgsodiumUserRepository) UpdatePassword(ctx context.Context, userID, passwordHash string) error {

	_, err := r.db.ExecContext(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", passwordHash, userID)

	if err != nil {

		return apperrors.Internal("failed to update password", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) UpdateProfilePhoto(ctx context.Context, userID, photoURL string) error {

	_, err := r.db.ExecContext(ctx, "UPDATE users SET profile_photo_url = $1, updated_at = NOW() WHERE id = $2", photoURL, userID)

	if err != nil {

		return apperrors.Internal(errFailedToUpdateProfilePhoto, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) RemoveProfilePhoto(ctx context.Context, userID string) error {

	_, err := r.db.ExecContext(ctx, "UPDATE users SET profile_photo_url = NULL, updated_at = NOW() WHERE id = $1", userID)

	if err != nil {

		return apperrors.Internal(errFailedToUpdateProfilePhoto, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) GetTOTPState(ctx context.Context, userID string) ([]byte, bool, []string, int32, error) {

	var encryptedSecret []byte

	var totpEnabled bool

	var backupCodesHash []string

	var backupCodesRemaining int32

	err := r.db.QueryRowContext(ctx, `

		SELECT totp_secret_encrypted, totp_enabled, totp_backup_codes_hash, totp_backup_codes_remaining

		FROM users WHERE id = $1

	`, userID).Scan(&encryptedSecret, &totpEnabled, pq.Array(&backupCodesHash), &backupCodesRemaining)

	if errors.Is(err, sql.ErrNoRows) {

		return nil, false, nil, 0, apperrors.NotFound(errUserNotFound)

	}

	if err != nil {

		return nil, false, nil, 0, apperrors.Internal(errFailedToGetTOTPState, err)

	}

	return encryptedSecret, totpEnabled, backupCodesHash, backupCodesRemaining, nil

}

func (r *PgsodiumUserRepository) EnableTOTP(ctx context.Context, userID string, encryptedSecret []byte, hashedCodes []string, backupCodesRemaining int32) error {

	_, err := r.db.ExecContext(ctx, `

		UPDATE users 

		SET totp_secret_encrypted = $1, totp_enabled = true, totp_backup_codes_hash = $2, totp_backup_codes_remaining = $3, updated_at = NOW()

		WHERE id = $4

	`, encryptedSecret, pq.Array(hashedCodes), backupCodesRemaining, userID)

	if err != nil {

		return apperrors.Internal(errFailedToEnableTOTP, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) RemoveBackupCode(ctx context.Context, userID string, codeHash string) error {

	_, err := r.db.ExecContext(ctx, `

		UPDATE users 

		SET totp_backup_codes_hash = array_remove(totp_backup_codes_hash, $1),

		    totp_backup_codes_remaining = GREATEST(totp_backup_codes_remaining - 1, 0)

		WHERE id = $2

	`, codeHash, userID)

	if err != nil {

		return apperrors.Internal(errFailedToRemoveBackupCode, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) DisableTOTP(ctx context.Context, userID string) error {

	_, err := r.db.ExecContext(ctx, `

		UPDATE users 

		SET totp_secret_encrypted = NULL, totp_enabled = false, 

		    totp_backup_codes_hash = NULL, totp_backup_codes_remaining = 0, updated_at = NOW()

		WHERE id = $1

	`, userID)

	if err != nil {

		return apperrors.Internal(errFailedToDisableTOTP, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) DeleteProfileData(ctx context.Context, userID string) error {

	_, err := r.db.ExecContext(ctx, `

		UPDATE users SET

			email_encrypted = NULL,

			email_hash = NULL,

			full_name_encrypted = NULL,

			full_name_nonce = NULL,

			full_name_hash = NULL,

			nickname_encrypted = NULL,

			nickname_nonce = NULL,

			nickname_hash = NULL,

			profile_photo_url = NULL,

			password_hash = NULL,

			email_confirmed = FALSE,

			updated_at = NOW()

		WHERE id = $1

	`, userID)

	if err != nil {

		return apperrors.Internal(errFailedToDeleteProfile, err)

	}

	return nil

}

func (r *PgsodiumUserRepository) ReplaceUserGoals(ctx context.Context, userID string, goals []string) error {

	if err := r.deleteListItems(ctx, userID, "user_goals"); err != nil {

		return err

	}

	for _, goal := range goals {

		if err := r.insertListItem(ctx, userID, "user_goals", "goal", goal); err != nil {

			return err

		}

	}

	return nil

}

func (r *PgsodiumUserRepository) ReplaceUserContraindications(ctx context.Context, userID string, contraindications []string) error {

	if err := r.deleteListItems(ctx, userID, "user_contraindications"); err != nil {

		return err

	}

	for _, contra := range contraindications {

		if err := r.insertListItem(ctx, userID, "user_contraindications", "contraindication", contra); err != nil {

			return err

		}

	}

	return nil

}

func (r *PgsodiumUserRepository) deleteListItems(ctx context.Context, userID, tableName string) error {

	_, err := r.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE user_id = $1", tableName), userID) // NOSONAR go:S2077 - tableName is a hardcoded allow-listed constant, not user input

	if err != nil {

		return apperrors.Internal("failed to delete list items", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) insertListItem(ctx context.Context, userID, tableName, columnName, value string) error {

	query := fmt.Sprintf("INSERT INTO %s (user_id, %s) VALUES ($1, $2) ON CONFLICT DO NOTHING", tableName, columnName) // nolint:gosec // tableName and columnName are hardcoded allow-listed constants, not user input // NOSONAR go:S2077 // nosem

	_, err := r.db.ExecContext(ctx, query, userID, value) // NOSONAR go:S2077

	if err != nil {

		return apperrors.Internal("failed to insert list item", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) DeleteHealthCondition(ctx context.Context, id, userID string) error {

	_, err := r.db.ExecContext(ctx, "DELETE FROM user_health_conditions WHERE id = $1 AND user_id = $2", id, userID)

	if err != nil {

		return apperrors.Internal("failed to delete health condition", err)

	}

	return nil

}

func (r *PgsodiumUserRepository) DeleteMenstrualCycle(ctx context.Context, id, userID string) error {

	_, err := r.db.ExecContext(ctx, "DELETE FROM user_menstrual_cycles WHERE id = $1 AND user_id = $2", id, userID)

	if err != nil {

		return apperrors.Internal("failed to delete menstrual cycle", err)

	}

	return nil

}
