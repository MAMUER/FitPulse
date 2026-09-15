// Package postgres provides PostgreSQL repository implementations.

package postgres

// TrainingProgress holds training progress metrics.

type TrainingProgress struct {
	TotalPlans int `json:"total_plans"`

	CompletedWorkouts int `json:"completed_workouts"`

	CompletionRate float64 `json:"completion_rate"`
}

// ScanTrainingProgress scans progress query results.

func ScanTrainingProgress(totalPlans, completedWorkouts int) TrainingProgress {

	return TrainingProgress{

		TotalPlans: totalPlans,

		CompletedWorkouts: completedWorkouts,

		CompletionRate: float64(completedWorkouts) / float64(totalPlans) * 100,
	}

}
