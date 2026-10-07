package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type externalSecretsRepository struct {
	db *sql.DB
}

func NewExternalSecretsRepository(db *sql.DB) port.ExternalSecretRepository {
	return &externalSecretsRepository{db: db}
}

func (r *externalSecretsRepository) GetByName(ctx context.Context, name string) (*port.ExternalSecret, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, value, created_at, updated_at
		FROM external_secrets
		WHERE name = $1
	`, name)

	secret := &port.ExternalSecret{}
	err := row.Scan(&secret.ID, &secret.Name, &secret.Value, &secret.CreatedAt, &secret.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("external secret not found")
		}
		return nil, apperrors.Internal("failed to get external secret", err)
	}
	return secret, nil
}

func (r *externalSecretsRepository) List(ctx context.Context) ([]*port.ExternalSecret, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, value, created_at, updated_at
		FROM external_secrets
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, apperrors.Internal("failed to list external secrets", err)
	}
	defer func() { _ = rows.Close() }()

	var secrets []*port.ExternalSecret
	for rows.Next() {
		secret := &port.ExternalSecret{}
		if err := rows.Scan(&secret.ID, &secret.Name, &secret.Value, &secret.CreatedAt, &secret.UpdatedAt); err != nil {
			return nil, apperrors.Internal("failed to scan external secret", err)
		}
		secrets = append(secrets, secret)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate external secrets", err)
	}
	return secrets, nil
}

func (r *externalSecretsRepository) Upsert(ctx context.Context, secret *port.ExternalSecret) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO external_secrets (name, value)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = NOW()
	`, secret.Name, secret.Value)
	if err != nil {
		return apperrors.Internal("failed to upsert external secret", err)
	}
	return nil
}

func (r *externalSecretsRepository) Delete(ctx context.Context, name string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM external_secrets WHERE name = $1`, name)
	if err != nil {
		return apperrors.Internal("failed to delete external secret", err)
	}
	return nil
}
