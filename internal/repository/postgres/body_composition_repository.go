package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type UserBodyCompositionRepository interface {
	List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*port.UserBodyComposition, error)
	Create(ctx context.Context, bc *port.UserBodyComposition) (*port.UserBodyComposition, error)
}

type userBodyCompositionRepository struct {
	db *sql.DB
}

func NewUserBodyCompositionRepository(db *sql.DB) port.UserBodyCompositionRepository {
	return &userBodyCompositionRepository{db: db}
}

func (r *userBodyCompositionRepository) List(ctx context.Context, userID string, from, to *time.Time, limit int) ([]*port.UserBodyComposition, error) {
	query, args := BuildDateRangeQuery(
		"user_body_composition",
		"id, user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source, created_at",
		"user_id",
		userID,
		from,
		to,
		limit,
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal("failed to list body composition", err)
	}
	defer func() { _ = rows.Close() }()

	records, err := ScanRows(rows, func(rows *sql.Rows) (*port.UserBodyComposition, error) {
		bc := &port.UserBodyComposition{}
		if err := rows.Scan(
			&bc.ID, &bc.UserID, &bc.RecordedAt, &bc.WeightKG, &bc.HeightCM, &bc.BMI,
			&bc.BodyFatPercentage, &bc.MuscleMassPercentage, &bc.BoneMassPercentage,
			&bc.WaterPercentage, &bc.VisceralFatRating, &bc.MetabolicAge, &bc.Source, &bc.CreatedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan body composition", err)
		}
		return bc, nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (r *userBodyCompositionRepository) Create(ctx context.Context, bc *port.UserBodyComposition) (*port.UserBodyComposition, error) {
	query := `
		INSERT INTO user_body_composition (user_id, recorded_at, weight_kg, height_cm, bmi, body_fat_percentage, muscle_mass_percentage, bone_mass_percentage, water_percentage, visceral_fat_rating, metabolic_age, source)
		VALUES ($1, COALESCE($2, NOW()), $3, $4, $5, $6, $7, $8, $9, $10, $11, COALESCE($12, 'manual'))
		RETURNING id, recorded_at
	`
	var recordedAt interface{}
	if !bc.RecordedAt.IsZero() {
		recordedAt = bc.RecordedAt
	} else {
		recordedAt = nil
	}
	var source interface{}
	if bc.Source == "" {
		source = nil
	} else {
		source = bc.Source
	}
	err := r.db.QueryRowContext(ctx, query,
		bc.UserID, recordedAt, bc.WeightKG, bc.HeightCM, bc.BMI,
		bc.BodyFatPercentage, bc.MuscleMassPercentage, bc.BoneMassPercentage,
		bc.WaterPercentage, bc.VisceralFatRating, bc.MetabolicAge, source,
	).Scan(&bc.ID, &bc.RecordedAt)
	if err != nil {
		return nil, apperrors.Internal("failed to create body composition", err)
	}
	return bc, nil
}
