// Package repository provides PostgreSQL implementation of medical constraints repository.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MAMUER/project/internal/biometric/domain"
	"github.com/MAMUER/project/internal/biometric/medical"
)

// postgresMedicalRepository implements medical.MedicalRepository using PostgreSQL.
type postgresMedicalRepository struct {
	db *sql.DB
}

// NewMedicalRepository creates a new PostgreSQL medical constraints repository.
func NewMedicalRepository(db *sql.DB) medical.MedicalRepository {
	return &postgresMedicalRepository{db: db}
}

// scanMedicalConstraint scans a single row into a MedicalConstraint.
func scanMedicalConstraint(rows *sql.Rows) (*domain.MedicalConstraint, error) {
	var c domain.MedicalConstraint
	var impactJSON []byte
	var validatedBy sql.NullString
	var validatedAt sql.NullTime

	if err := rows.Scan(
		&c.ID, &c.UserID, &c.Code, &c.Label, &c.Category, &c.Severity, &c.CustomText,
		&impactJSON, &validatedBy, &validatedAt, &c.Active, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan constraint: %w", err)
	}

	if validatedBy.Valid {
		v := validatedBy.String
		c.ValidatedBy = &v
	}
	if validatedAt.Valid {
		t := validatedAt.Time
		c.ValidatedAt = &t
	}

	if len(impactJSON) > 0 {
		if err := json.Unmarshal(impactJSON, &c.ImpactOnTraining); err != nil {
			return nil, fmt.Errorf("unmarshal impact rules: %w", err)
		}
	}

	return &c, nil
}

// scanMedicalConstraints scans all rows into a slice of MedicalConstraint.
func scanMedicalConstraints(rows *sql.Rows) ([]domain.MedicalConstraint, error) {
	var constraints []domain.MedicalConstraint
	for rows.Next() {
		c, err := scanMedicalConstraint(rows)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, *c)
	}
	return constraints, nil
}

// GetActiveConstraints retrieves all active medical constraints for a user.
func (r *postgresMedicalRepository) GetActiveConstraints(ctx context.Context, userID string) ([]domain.MedicalConstraint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, code, label, category, severity, custom_text,
		       impact_on_training, validated_by, validated_at, active, created_at, updated_at
		FROM medical_constraints
		WHERE user_id = $1 AND active = true
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query active constraints: %w", err)
	}
	defer func() { _ = rows.Close() }()

	constraints, err := scanMedicalConstraints(rows)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate constraints: %w", err)
	}

	return constraints, nil
}

// SaveConstraint persists a medical constraint.
func (r *postgresMedicalRepository) SaveConstraint(ctx context.Context, constraint domain.MedicalConstraint) error {
	if constraint.ID == "" {
		constraint.ID = fmt.Sprintf("mc-%d", time.Now().UnixNano())
	}

	impactJSON, err := json.Marshal(constraint.ImpactOnTraining)
	if err != nil {
		return fmt.Errorf("marshal impact rules: %w", err)
	}

	now := time.Now()
	if constraint.CreatedAt.IsZero() {
		constraint.CreatedAt = now
	}
	constraint.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO medical_constraints
		    (id, user_id, code, label, category, severity, custom_text,
		     impact_on_training, validated_by, validated_at, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			label = EXCLUDED.label,
			category = EXCLUDED.category,
			severity = EXCLUDED.severity,
			custom_text = EXCLUDED.custom_text,
			impact_on_training = EXCLUDED.impact_on_training,
			validated_by = EXCLUDED.validated_by,
			validated_at = EXCLUDED.validated_at,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at
	`, constraint.ID, constraint.UserID, constraint.Code, constraint.Label, constraint.Category,
		constraint.Severity, constraint.CustomText, impactJSON, constraint.ValidatedBy,
		constraint.ValidatedAt, constraint.Active, constraint.CreatedAt, constraint.UpdatedAt)

	if err != nil {
		return fmt.Errorf("save constraint: %w", err)
	}

	return nil
}

// DeleteConstraint deletes a medical constraint by ID.
func (r *postgresMedicalRepository) DeleteConstraint(ctx context.Context, constraintID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM medical_constraints WHERE id = $1
	`, constraintID)
	if err != nil {
		return fmt.Errorf("delete constraint: %w", err)
	}
	return nil
}

// GetConstraintByCode retrieves constraints by ICD-10 code.
func (r *postgresMedicalRepository) GetConstraintByCode(ctx context.Context, code string) ([]domain.MedicalConstraint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, code, label, category, severity, custom_text,
		       impact_on_training, validated_by, validated_at, active, created_at, updated_at
		FROM medical_constraints
		WHERE code = $1 AND active = true
		ORDER BY created_at DESC
	`, code)
	if err != nil {
		return nil, fmt.Errorf("query constraints by code: %w", err)
	}
	defer func() { _ = rows.Close }()

	constraints, err := scanMedicalConstraints(rows)
	if err != nil {
		return nil, err
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate constraints: %w", err)
	}

	return constraints, nil
}
