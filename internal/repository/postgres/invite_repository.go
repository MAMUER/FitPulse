package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type InviteCodeRepository interface {
	List(ctx context.Context, page, pageSize int) ([]*port.InviteCode, int, error)
	Create(ctx context.Context, invite *port.InviteCode) error
	Revoke(ctx context.Context, code string) error
	Validate(ctx context.Context, code string) (*port.InviteCode, error)
	ValidateInviteCodeUse(ctx context.Context, code string) (bool, string, string, string, error)
	LogInviteCodeUse(ctx context.Context, code, userID string) error
}

type inviteCodeRepository struct {
	db *sql.DB
}

func NewInviteCodeRepository(db *sql.DB) port.InviteCodeRepository {
	return &inviteCodeRepository{db: db}
}

func (r *inviteCodeRepository) List(ctx context.Context, page, pageSize int) ([]*port.InviteCode, int, error) {
	offset := page * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT code, role, specialty, max_uses, used_count, is_active, created_by, created_at,
		       COUNT(*) OVER() AS total_count
		FROM invite_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list invite codes", err)
	}
	defer func() { _ = rows.Close() }()

	var invites []*port.InviteCode
	var totalCount int
	for rows.Next() {
		inv := &port.InviteCode{}
		var specialty sql.NullString
		if scanErr := rows.Scan(&inv.Code, &inv.Role, &specialty, &inv.MaxUses, &inv.UsedCount, &inv.IsActive, &inv.CreatedBy, &inv.CreatedAt, &totalCount); scanErr != nil {
			return nil, 0, apperrors.Internal("failed to scan invite code", scanErr)
		}
		if specialty.Valid {
			s := specialty.String
			inv.Specialty = &s
		}
		invites = append(invites, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, apperrors.Internal("failed to iterate invite codes", err)
	}

	return invites, totalCount, nil
}

func (r *inviteCodeRepository) Create(ctx context.Context, invite *port.InviteCode) error {
	var specialty interface{}
	if invite.Specialty != nil {
		specialty = *invite.Specialty
	}
	query := `
		INSERT INTO invite_codes (code, role, specialty, max_uses, created_by, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, NOW())
	`
	_, err := r.db.ExecContext(ctx, query,
		invite.Code, invite.Role, specialty, invite.MaxUses, invite.CreatedBy,
	)
	if err != nil {
		return apperrors.Internal("failed to create invite code", err)
	}
	return nil
}

func (r *inviteCodeRepository) Revoke(ctx context.Context, code string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE invite_codes SET is_active = FALSE WHERE code = $1`, code)
	if err != nil {
		return apperrors.Internal("failed to revoke invite code", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.NotFound("invite code not found")
	}
	return nil
}

func (r *inviteCodeRepository) Validate(ctx context.Context, code string) (*port.InviteCode, error) {
	query := `
		SELECT code, role, specialty, max_uses, used_count, is_active, created_by, created_at
		FROM invite_codes
		WHERE code = $1 AND is_active = TRUE
	`
	inv := &port.InviteCode{}
	var specialty sql.NullString
	if err := r.db.QueryRowContext(ctx, query, code).Scan(
		&inv.Code, &inv.Role, &specialty, &inv.MaxUses, &inv.UsedCount, &inv.IsActive, &inv.CreatedBy, &inv.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("invite code not found or inactive")
		}
		return nil, apperrors.Internal("failed to validate invite code", err)
	}
	if specialty.Valid {
		s := specialty.String
		inv.Specialty = &s
	}
	return inv, nil
}

func (r *inviteCodeRepository) UseInviteCode(ctx context.Context, code string) error {
	result, err := r.db.ExecContext(ctx, `SELECT * FROM use_invite_code($1)`, code)
	if err != nil {
		return apperrors.Internal("failed to use invite code", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return apperrors.NotFound("invite code not found or invalid")
	}
	return nil
}

func (r *inviteCodeRepository) ValidateInviteCodeUse(ctx context.Context, code string) (bool, string, string, string, error) {
	var isValid bool
	var role, specialty, errMsg sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT * FROM use_invite_code($1)`, code).Scan(&isValid, &role, &specialty, &errMsg)
	if err != nil {
		return false, "", "", "", apperrors.Internal("failed to validate invite code", err)
	}
	roleStr := ""
	if role.Valid {
		roleStr = role.String
	}
	specialtyStr := ""
	if specialty.Valid {
		specialtyStr = specialty.String
	}
	errMsgStr := ""
	if errMsg.Valid {
		errMsgStr = errMsg.String
	}
	return isValid, roleStr, specialtyStr, errMsgStr, nil
}

func (r *inviteCodeRepository) LogInviteCodeUse(ctx context.Context, code, userID string) error {
	_, err := r.db.ExecContext(ctx, `SELECT log_invite_code_use($1, $2)`, code, userID)
	if err != nil {
		return apperrors.Internal("failed to log invite code use", err)
	}
	return nil
}
