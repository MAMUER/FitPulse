package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type UserMenstrualRepository interface {
	ListCycles(ctx context.Context, userID string) ([]*port.UserMenstrualCycle, error)
	CreateCycle(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error)
	UpdateCycle(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error)
	DeleteCycle(ctx context.Context, id, userID string) error
	ListSymptoms(ctx context.Context, cycleID string) ([]string, error)
	CreateSymptom(ctx context.Context, cycleID, symptom string) error
	DeleteSymptoms(ctx context.Context, cycleID string) error
	ListMoods(ctx context.Context, cycleID string) ([]string, error)
	CreateMood(ctx context.Context, cycleID, mood string) error
	DeleteMoods(ctx context.Context, cycleID string) error
}

type userMenstrualRepository struct {
	db *sql.DB
}

const (
	errFailedToListMenstrualCycles     = "failed to list menstrual cycles"
	errFailedToCreateMenstrualCycle    = "failed to create menstrual cycle"
	errFailedToUpdateMenstrualCycle    = "failed to update menstrual cycle"
	errFailedToDeleteMenstrualCycle    = "failed to delete menstrual cycle"
	errFailedToListMenstrualSymptoms   = "failed to list menstrual symptoms"
	errFailedToCreateMenstrualSymptom  = "failed to create menstrual symptom"
	errFailedToDeleteMenstrualSymptoms = "failed to delete menstrual symptoms"
	errFailedToListMenstrualMoods      = "failed to list menstrual moods"
	errFailedToCreateMenstrualMood     = "failed to create menstrual mood"
	errFailedToDeleteMenstrualMoods    = "failed to delete menstrual moods"
	errFailedToBeginTransaction        = "failed to begin transaction"
	errFailedToCommitTransaction       = "failed to commit transaction"
	errFailedToScanMenstrualCycle      = "failed to scan menstrual cycle"
	errFailedToIterateMenstrualCycles  = "failed to iterate menstrual cycles"
)

func NewUserMenstrualRepository(db *sql.DB) port.UserMenstrualRepository {
	return &userMenstrualRepository{db: db}
}

func (r *userMenstrualRepository) ListCycles(ctx context.Context, userID string) ([]*port.UserMenstrualCycle, error) {
	query := `
		SELECT id, user_id, cycle_start_date, cycle_end_date, flow_intensity, notes, created_at, updated_at
		FROM user_menstrual_cycles
		WHERE user_id = $1
		ORDER BY cycle_start_date DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Internal(errFailedToListMenstrualCycles, err)
	}
	defer func() { _ = rows.Close() }()

	var cycles []*port.UserMenstrualCycle
	for rows.Next() {
		var c port.UserMenstrualCycle
		var cycleEndDate sql.NullString
		var notes sql.NullString
		if err := rows.Scan(&c.ID, &c.UserID, &c.CycleStartDate, &cycleEndDate, &c.FlowIntensity, &notes, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, apperrors.Internal(errFailedToScanMenstrualCycle, err)
		}
		if cycleEndDate.Valid {
			c.CycleEndDate = cycleEndDate.String
		}
		if notes.Valid {
			c.Notes = notes.String
		}
		cycles = append(cycles, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal(errFailedToIterateMenstrualCycles, err)
	}
	return cycles, nil
}

func (r *userMenstrualRepository) CreateCycle(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	query := `
		INSERT INTO user_menstrual_cycles (user_id, cycle_start_date, cycle_end_date, flow_intensity, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var cycleEndDate interface{}
	if cycle.CycleEndDate != "" {
		cycleEndDate = cycle.CycleEndDate
	} else {
		cycleEndDate = nil
	}
	var flowIntensity interface{}
	if cycle.FlowIntensity != "" {
		flowIntensity = cycle.FlowIntensity
	} else {
		flowIntensity = nil
	}
	err := r.db.QueryRowContext(ctx, query,
		cycle.UserID, cycle.CycleStartDate, cycleEndDate, flowIntensity, cycle.Notes,
	).Scan(&cycle.ID)
	if err != nil {
		return nil, apperrors.Internal(errFailedToCreateMenstrualCycle, err)
	}
	return cycle, nil
}

func (r *userMenstrualRepository) UpdateCycle(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	query := `
		UPDATE user_menstrual_cycles
		SET cycle_end_date = $1, flow_intensity = $2, notes = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
	`
	var cycleEndDate interface{}
	if cycle.CycleEndDate != "" {
		cycleEndDate = cycle.CycleEndDate
	} else {
		cycleEndDate = nil
	}
	var flowIntensity interface{}
	if cycle.FlowIntensity != "" {
		flowIntensity = cycle.FlowIntensity
	} else {
		flowIntensity = nil
	}
	_, err := r.db.ExecContext(ctx, query, cycleEndDate, flowIntensity, cycle.Notes, cycle.ID, cycle.UserID)
	if err != nil {
		return nil, apperrors.Internal(errFailedToUpdateMenstrualCycle, err)
	}
	return cycle, nil
}

func (r *userMenstrualRepository) DeleteCycle(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_menstrual_cycles WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return apperrors.Internal(errFailedToDeleteMenstrualCycle, err)
	}
	return nil
}

func (r *userMenstrualRepository) ListSymptoms(ctx context.Context, cycleID string) ([]string, error) {
	return r.listStrings(ctx, `SELECT symptom FROM user_menstrual_symptoms WHERE cycle_id = $1`, errFailedToListMenstrualSymptoms, cycleID)
}

func (r *userMenstrualRepository) ListMoods(ctx context.Context, cycleID string) ([]string, error) {
	return r.listStrings(ctx, `SELECT mood FROM user_menstrual_moods WHERE cycle_id = $1`, errFailedToListMenstrualMoods, cycleID)
}

func (r *userMenstrualRepository) listStrings(ctx context.Context, query, errMsg string, args ...interface{}) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, apperrors.Internal(errMsg, err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]string, 0)
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err == nil {
			items = append(items, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal(errMsg, err)
	}
	return items, nil
}

func (r *userMenstrualRepository) CreateSymptom(ctx context.Context, cycleID, symptom string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_menstrual_symptoms (cycle_id, symptom) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		cycleID, symptom,
	)
	if err != nil {
		return apperrors.Internal(errFailedToCreateMenstrualSymptom, err)
	}
	return nil
}

func (r *userMenstrualRepository) DeleteSymptoms(ctx context.Context, cycleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_menstrual_symptoms WHERE cycle_id = $1`, cycleID)
	if err != nil {
		return apperrors.Internal(errFailedToDeleteMenstrualSymptoms, err)
	}
	return nil
}

func (r *userMenstrualRepository) CreateMood(ctx context.Context, cycleID, mood string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO user_menstrual_moods (cycle_id, mood) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		cycleID, mood,
	)
	if err != nil {
		return apperrors.Internal(errFailedToCreateMenstrualMood, err)
	}
	return nil
}

func (r *userMenstrualRepository) DeleteMoods(ctx context.Context, cycleID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_menstrual_moods WHERE cycle_id = $1`, cycleID)
	if err != nil {
		return apperrors.Internal(errFailedToDeleteMenstrualMoods, err)
	}
	return nil
}

func (r *userMenstrualRepository) CreateCycleWithDetails(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.Internal(errFailedToBeginTransaction, err)
	}
	defer func() { _ = tx.Rollback() }()

	var cycleEndDate interface{}
	if cycle.CycleEndDate != "" {
		cycleEndDate = cycle.CycleEndDate
	}
	var flowIntensity interface{}
	if cycle.FlowIntensity != "" {
		flowIntensity = cycle.FlowIntensity
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO user_menstrual_cycles (user_id, cycle_start_date, cycle_end_date, flow_intensity, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, cycle.UserID, cycle.CycleStartDate, cycleEndDate, flowIntensity, cycle.Notes).Scan(&cycle.ID)
	if err != nil {
		return nil, apperrors.Internal(errFailedToCreateMenstrualCycle, err)
	}

	for _, symptom := range cycle.Symptoms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_symptoms (cycle_id, symptom) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cycle.ID, symptom); err != nil {
			return nil, apperrors.Internal(errFailedToCreateMenstrualSymptom, err)
		}
	}
	for _, mood := range cycle.Moods {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_moods (cycle_id, mood) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cycle.ID, mood); err != nil {
			return nil, apperrors.Internal(errFailedToCreateMenstrualMood, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.Internal(errFailedToCommitTransaction, err)
	}
	return cycle, nil
}

func (r *userMenstrualRepository) UpdateCycleWithDetails(ctx context.Context, cycle *port.UserMenstrualCycle) (*port.UserMenstrualCycle, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, apperrors.Internal(errFailedToBeginTransaction, err)
	}
	defer func() { _ = tx.Rollback() }()

	var cycleEndDate interface{}
	if cycle.CycleEndDate != "" {
		cycleEndDate = cycle.CycleEndDate
	}
	var flowIntensity interface{}
	if cycle.FlowIntensity != "" {
		flowIntensity = cycle.FlowIntensity
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE user_menstrual_cycles
		SET cycle_end_date = $1, flow_intensity = $2, notes = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
	`, cycleEndDate, flowIntensity, cycle.Notes, cycle.ID, cycle.UserID)
	if err != nil {
		return nil, apperrors.Internal(errFailedToUpdateMenstrualCycle, err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_menstrual_symptoms WHERE cycle_id = $1`, cycle.ID); err != nil {
		return nil, apperrors.Internal(errFailedToDeleteMenstrualSymptoms, err)
	}
	for _, symptom := range cycle.Symptoms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_symptoms (cycle_id, symptom) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cycle.ID, symptom); err != nil {
			return nil, apperrors.Internal(errFailedToCreateMenstrualSymptom, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_menstrual_moods WHERE cycle_id = $1`, cycle.ID); err != nil {
		return nil, apperrors.Internal(errFailedToDeleteMenstrualMoods, err)
	}
	for _, mood := range cycle.Moods {
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_menstrual_moods (cycle_id, mood) VALUES ($1, $2) ON CONFLICT DO NOTHING`, cycle.ID, mood); err != nil {
			return nil, apperrors.Internal(errFailedToCreateMenstrualMood, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, apperrors.Internal(errFailedToCommitTransaction, err)
	}
	return cycle, nil
}
