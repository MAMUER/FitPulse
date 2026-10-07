package pgx

import (
	"context"
	"encoding/json"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

// SurveyRepositoryPGX implements survey operations using pgxpool.Pool.
type SurveyRepositoryPGX struct {
	db DB
}

// NewSurveyRepositoryPGX creates a new survey repository.
func NewSurveyRepositoryPGX(db DB) *SurveyRepositoryPGX {
	return &SurveyRepositoryPGX{db: db}
}

// SaveSurvey saves survey data to user_profiles.
func (r *SurveyRepositoryPGX) SaveSurvey(ctx context.Context, userID string, survey map[string]interface{}, completed bool, completedAt *time.Time) error {
	surveyJSON, err := json.Marshal(survey)
	if err != nil {
		return apperrors.Internal("failed to marshal survey", err)
	}

	query := `
		UPDATE user_profiles
		SET survey_data = $1, survey_completed = $2, survey_completed_at = $3, updated_at = NOW()
		WHERE user_id = $4
	`
	_, err = r.db.Exec(ctx, query, surveyJSON, completed, completedAt, userID)
	if err != nil {
		return apperrors.Internal("failed to save survey", err)
	}
	return nil
}

// LoadSurvey loads survey data from user_profiles.
func (r *SurveyRepositoryPGX) LoadSurvey(ctx context.Context, userID string) (map[string]interface{}, bool, *time.Time, error) {
	var surveyJSON []byte
	var completed bool
	var completedAt *time.Time

	err := r.db.QueryRow(ctx, `
		SELECT survey_data, survey_completed, survey_completed_at
		FROM user_profiles
		WHERE user_id = $1
	`, userID).Scan(&surveyJSON, &completed, &completedAt)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return map[string]interface{}{}, false, nil, nil
		}
		return nil, false, nil, apperrors.Internal("failed to load survey", err)
	}

	if len(surveyJSON) == 0 {
		return map[string]interface{}{}, false, nil, nil
	}

	var survey map[string]interface{}
	if err := json.Unmarshal(surveyJSON, &survey); err != nil {
		return nil, false, nil, apperrors.Internal("failed to unmarshal survey", err)
	}

	return survey, completed, completedAt, nil
}

// Ensure SurveyRepositoryPGX implements port.SurveyRepository.
var _ port.SurveyRepository = (*SurveyRepositoryPGX)(nil)
