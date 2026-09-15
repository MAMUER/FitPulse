// Package shared contains reusable training queries and scanner helpers.
package shared

const (
	TrainingPlanColumns      = "id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at"
	TrainingPlanColumnsTotal = "id, user_id, classification, duration_weeks, available_days, plan_data, created_at, updated_at, COUNT(*) OVER() AS total_count"

	QueryGetPlan     = "SELECT " + TrainingPlanColumns + " FROM training_plans WHERE id = $1 AND user_id = $2"
	QueryListPlans   = "SELECT " + TrainingPlanColumnsTotal + " FROM training_plans WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	QueryGetProgress = "SELECT COUNT(*) as total_plans, COUNT(CASE WHEN updated_at > created_at THEN 1 END) as completed_workouts FROM training_plans WHERE user_id = $1"
	QueryGetAchievements = "SELECT id, user_id, type, title, description, earned_at FROM achievements WHERE user_id = $1 ORDER BY earned_at DESC"
)
