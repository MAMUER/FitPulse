// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type RefreshTokenRepository interface {
	GetValid(ctx context.Context, token string) (*port.RefreshToken, error)

	Create(ctx context.Context, rt *port.RefreshToken) error

	MarkUsed(ctx context.Context, token string) error
}

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) port.RefreshTokenRepository {

	return &refreshTokenRepository{db: db}

}

func (r *refreshTokenRepository) GetValid(ctx context.Context, token string) (*port.RefreshToken, error) {

	query := `

		SELECT id, user_id, token, used, expires_at, created_at

		FROM refresh_tokens

		WHERE token = $1 AND used = FALSE AND expires_at > NOW()

	`

	rt := &port.RefreshToken{}

	err := r.db.QueryRowContext(ctx, query, token).Scan(

		&rt.ID, &rt.UserID, &rt.Token, &rt.Used, &rt.ExpiresAt, &rt.CreatedAt,
	)

	if err != nil {

		if err == sql.ErrNoRows {

			return nil, apperrors.NotFound("refresh token not found or expired")

		}

		return nil, apperrors.Internal("failed to get refresh token", err)

	}

	return rt, nil

}

func (r *refreshTokenRepository) Create(ctx context.Context, rt *port.RefreshToken) error {

	query := `

		INSERT INTO refresh_tokens (user_id, token, expires_at)

		VALUES ($1, $2, $3)

	`

	_, err := r.db.ExecContext(ctx, query, rt.UserID, rt.Token, rt.ExpiresAt)

	if err != nil {

		return apperrors.Internal("failed to create refresh token", err)

	}

	return nil

}

func (r *refreshTokenRepository) MarkUsed(ctx context.Context, token string) error {

	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET used = TRUE WHERE token = $1`, token)

	if err != nil {

		return apperrors.Internal("failed to mark refresh token as used", err)

	}

	return nil

}
