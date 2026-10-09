package shared

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/domain/entity"
)

func TestScanTrainingProgress(t *testing.T) {
	t.Run("computes completion rate", func(t *testing.T) {
		progress := ScanTrainingProgress(10, 5)
		assert.Equal(t, 10, progress.TotalPlans)
		assert.Equal(t, 5, progress.CompletedWorkouts)
		assert.Equal(t, 50.0, progress.CompletionRate)
	})

	t.Run("zero plans returns zero rate", func(t *testing.T) {
		progress := ScanTrainingProgress(0, 0)
		assert.Equal(t, 0, progress.TotalPlans)
		assert.Equal(t, 0, progress.CompletedWorkouts)
		assert.True(t, math.IsNaN(progress.CompletionRate), "NaN expected for 0/0")
	})

	t.Run("full completion returns 100 percent", func(t *testing.T) {
		progress := ScanTrainingProgress(7, 7)
		assert.Equal(t, 100.0, progress.CompletionRate)
	})
}

func TestScanTrainingPlanRows(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		rows := newMockTrainingRows([]*entity.TrainingPlan{
			{ID: "plan-1", UserID: "user-1", Classification: "strength", DurationWeeks: 4},
		}, 1)
		plans, totalCount, err := scanTrainingPlanRows(rows)
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "plan-1", plans[0].ID)
		assert.Equal(t, 1, totalCount)
	})

	t.Run("returns error on scan failure", func(t *testing.T) {
		rows := newMockTrainingRows([]*entity.TrainingPlan{{}}, 0)
		rows.scanErr = assert.AnError
		plans, _, err := scanTrainingPlanRows(rows)
		assert.Nil(t, plans)
		assert.Error(t, err)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		rows := newMockTrainingRows(nil, 0)
		rows.iterErr = assert.AnError
		plans, _, err := scanTrainingPlanRows(rows)
		assert.Nil(t, plans)
		assert.Error(t, err)
	})
}

func TestScanAchievementRows(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		rows := newMockAchievementRows([]*entity.Achievement{
			{ID: "ach-1", UserID: "user-1", Type: "milestone", Title: "First Workout", EarnedAt: time.Now()},
		})
		achievements, err := scanAchievementRows(rows)
		require.NoError(t, err)
		require.Len(t, achievements, 1)
		assert.Equal(t, "ach-1", achievements[0].ID)
		assert.Equal(t, "First Workout", achievements[0].Title)
	})

	t.Run("returns error on scan failure", func(t *testing.T) {
		rows := newMockAchievementRows([]*entity.Achievement{{}})
		rows.scanErr = assert.AnError
		achievements, err := scanAchievementRows(rows)
		assert.Nil(t, achievements)
		assert.Error(t, err)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		rows := newMockAchievementRows(nil)
		rows.iterErr = assert.AnError
		achievements, err := scanAchievementRows(rows)
		assert.Nil(t, achievements)
		assert.Error(t, err)
	})
}

type mockTrainingRows struct {
	plans   []*entity.TrainingPlan
	idx     int
	total   int
	scanErr error
	iterErr error
}

func newMockTrainingRows(plans []*entity.TrainingPlan, total int) *mockTrainingRows {
	return &mockTrainingRows{plans: plans, total: total}
}

func (m *mockTrainingRows) Next() bool {
	if m.idx < len(m.plans) {
		m.idx++
		return true
	}
	return false
}

func (m *mockTrainingRows) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	if m.idx-1 < len(m.plans) {
		plan := m.plans[m.idx-1]
		id := dest[0].(*string)
		userID := dest[1].(*string)
		classification := dest[2].(*string)
		durationWeeks := dest[3].(*int)
		availableDays := dest[4].(*[]int)
		planDataJSON := dest[5].(*[]byte)
		createdAt := dest[6].(*time.Time)
		updatedAt := dest[7].(*time.Time)
		totalCount := dest[8].(*int)
		*id = plan.ID
		*userID = plan.UserID
		*classification = plan.Classification
		*durationWeeks = plan.DurationWeeks
		*availableDays = plan.AvailableDays
		*createdAt = plan.CreatedAt
		*updatedAt = plan.UpdatedAt
		*totalCount = m.total
		if plan.PlanData != nil {
			data, _ := json.Marshal(plan.PlanData)
			*planDataJSON = data
		}
	}
	return nil
}

func (m *mockTrainingRows) Err() error {
	return m.iterErr
}

func (m *mockTrainingRows) Close() {}

type mockAchievementRows struct {
	achievements []*entity.Achievement
	idx          int
	scanErr      error
	iterErr      error
}

func newMockAchievementRows(achievements []*entity.Achievement) *mockAchievementRows {
	return &mockAchievementRows{achievements: achievements}
}

func (m *mockAchievementRows) Next() bool {
	if m.idx < len(m.achievements) {
		m.idx++
		return true
	}
	return false
}

func (m *mockAchievementRows) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	if m.idx-1 < len(m.achievements) {
		a := m.achievements[m.idx-1]
		id := dest[0].(*string)
		userID := dest[1].(*string)
		typ := dest[2].(*string)
		title := dest[3].(*string)
		description := dest[4].(*string)
		earnedAt := dest[5].(*time.Time)
		*id = a.ID
		*userID = a.UserID
		*typ = a.Type
		*title = a.Title
		*description = a.Description
		*earnedAt = a.EarnedAt
	}
	return nil
}

func (m *mockAchievementRows) Err() error {
	return m.iterErr
}

func (m *mockAchievementRows) Close() {}
