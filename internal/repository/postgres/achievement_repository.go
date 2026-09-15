// Package postgres provides PostgreSQL repository implementations.

package postgres

import (
	"context"
	"database/sql"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/port"
)

type AchievementRepositoryEx interface {
	ListWithEarnedStatus(ctx context.Context, userID string) ([]*port.AchievementInfo, error)

	Earn(ctx context.Context, userID, achievementID string) error
}

type achievementRepositoryEx struct {
	db *sql.DB
}

func NewAchievementRepositoryEx(db *sql.DB) port.AchievementRepositoryEx {

	return &achievementRepositoryEx{db: db}

}

func (r *achievementRepositoryEx) ListWithEarnedStatus(ctx context.Context, userID string) ([]*port.AchievementInfo, error) {

	query := `

		SELECT a.id, a.name, a.description, a.icon_url, ua.earned_at, a.created_at

		FROM achievements a

		LEFT JOIN user_achievements ua ON ua.achievement_id = a.id AND ua.user_id = $1

		ORDER BY a.created_at ASC

	`

	rows, err := r.db.QueryContext(ctx, query, userID)

	if err != nil {

		return nil, apperrors.Internal("failed to list achievements", err)

	}

	defer func() { _ = rows.Close() }()

	var achievements []*port.AchievementInfo

	for rows.Next() {

		var a port.AchievementInfo

		var earnedAt sql.NullTime

		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.IconURL, &earnedAt, &a.CreatedAt); err != nil {

			return nil, apperrors.Internal("failed to scan achievement", err)

		}

		if earnedAt.Valid {

			t := earnedAt.Time

			a.EarnedAt = &t

		}

		achievements = append(achievements, &a)

	}

	if err := rows.Err(); err != nil {

		return nil, apperrors.Internal("failed to iterate achievements", err)

	}

	return achievements, nil

}

func (r *achievementRepositoryEx) Earn(ctx context.Context, userID, achievementID string) error {

	query := `

		INSERT INTO user_achievements (user_id, achievement_id, earned_at)

		VALUES ($1, $2, NOW())

		ON CONFLICT DO NOTHING

	`

	_, err := r.db.ExecContext(ctx, query, userID, achievementID)

	if err != nil {

		return apperrors.Internal("failed to earn achievement", err)

	}

	return nil

}
