// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type UserHealthConditionRepository interface {
	List(ctx context.Context, userID string) ([]*port.UserHealthCondition, error)

	Upsert(ctx context.Context, condition *port.UserHealthCondition) (*port.UserHealthCondition, error)

	Delete(ctx context.Context, id, userID string) error
}

type userHealthConditionRepository struct {
	db *sql.DB
}

func NewUserHealthConditionRepository(db *sql.DB) port.UserHealthConditionRepository {

	return &userHealthConditionRepository{db: db}

}

func (r *userHealthConditionRepository) List(ctx context.Context, userID string) ([]*port.UserHealthCondition, error) {

	query := `

		SELECT id, user_id, condition_type, condition_name, severity, diagnosed_at, is_active, notes, created_at, updated_at

		FROM user_health_conditions

		WHERE user_id = $1

		ORDER BY created_at DESC

	`

	rows, err := r.db.QueryContext(ctx, query, userID)

	if err != nil {

		return nil, apperrors.Internal("failed to list health conditions", err)

	}

	defer func() { _ = rows.Close() }()

	var conditions []*port.UserHealthCondition

	for rows.Next() {

		var c port.UserHealthCondition

		var diagnosedAt sql.NullTime

		var notes sql.NullString

		if err := rows.Scan(&c.ID, &c.UserID, &c.ConditionType, &c.ConditionName, &c.Severity, &diagnosedAt, &c.IsActive, &notes, &c.CreatedAt, &c.UpdatedAt); err != nil {

			return nil, apperrors.Internal("failed to scan health condition", err)

		}

		if diagnosedAt.Valid {

			t := diagnosedAt.Time

			c.DiagnosedAt = &t

		}

		if notes.Valid {

			s := notes.String

			c.Notes = &s

		}

		conditions = append(conditions, &c)

	}

	if err := rows.Err(); err != nil {

		return nil, apperrors.Internal("failed to iterate health conditions", err)

	}

	return conditions, nil

}

func (r *userHealthConditionRepository) Upsert(ctx context.Context, condition *port.UserHealthCondition) (*port.UserHealthCondition, error) {

	query := `

		INSERT INTO user_health_conditions (user_id, condition_type, condition_name, severity, diagnosed_at, is_active, notes, updated_at)

		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())

		ON CONFLICT (user_id, condition_type, condition_name) DO UPDATE SET

			severity = EXCLUDED.severity,

			diagnosed_at = EXCLUDED.diagnosed_at,

			is_active = EXCLUDED.is_active,

			notes = EXCLUDED.notes,

			updated_at = NOW()

		RETURNING id

	`

	var diagnosedAt interface{}

	if condition.DiagnosedAt != nil {

		diagnosedAt = *condition.DiagnosedAt

	} else {

		diagnosedAt = nil

	}

	var notes interface{}

	if condition.Notes != nil {

		notes = *condition.Notes

	} else {

		notes = nil

	}

	err := r.db.QueryRowContext(ctx, query,

		condition.UserID, condition.ConditionType, condition.ConditionName,

		condition.Severity, diagnosedAt, condition.IsActive, notes,
	).Scan(&condition.ID)

	if err != nil {

		return nil, apperrors.Internal("failed to upsert health condition", err)

	}

	return condition, nil

}

func (r *userHealthConditionRepository) Delete(ctx context.Context, id, userID string) error {

	_, err := r.db.ExecContext(ctx, `DELETE FROM user_health_conditions WHERE id = $1 AND user_id = $2`, id, userID)

	if err != nil {

		return apperrors.Internal("failed to delete health condition", err)

	}

	return nil

}
