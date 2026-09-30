// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/db"
	"github.com/MAMUER/project/internal/domain/port"
)

type refreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) port.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) GetValid(ctx context.Context, token string) (*port.RefreshToken, error) {
	tokenHash := db.BlindIndex(token)

	query := `
		SELECT id, user_id, ` + db.PgsodiumDecryptParam("token_encrypted", "token_nonce", "token") + `, revoked, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked = FALSE AND expires_at > NOW()
	`

	rt := &port.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
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
	tokenHash := db.BlindIndex(rt.Token)
	nonce, err := db.GenerateNonce()
	if err != nil {
		return apperrors.Internal("failed to generate nonce", err)
	}

	query := `
		INSERT INTO refresh_tokens (token_hash, token_encrypted, token_nonce, user_id, expires_at)
		VALUES ($1, ` + db.PgsodiumRandomEncryptParam(2, 3) + `, $4, $5)
	`
	_, err = r.db.ExecContext(ctx, query, tokenHash, rt.Token, nonce, rt.UserID, rt.ExpiresAt)
	if err != nil {
		return apperrors.Internal("failed to create refresh token", err)
	}
	return nil
}

func (r *refreshTokenRepository) MarkUsed(ctx context.Context, token string) error {
	tokenHash := db.BlindIndex(token)
	_, err := r.db.ExecContext(ctx, `UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return apperrors.Internal("failed to mark refresh token as used", err)
	}
	return nil
}
