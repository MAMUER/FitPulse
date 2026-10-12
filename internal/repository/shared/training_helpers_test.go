package shared

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
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
			data, marshalErr := json.Marshal(plan.PlanData)
			if marshalErr != nil {
				panic("test helper: failed to marshal plan data: " + marshalErr.Error())
			}
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

func TestScanTrainingPlans(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		now := time.Now()
		rows := newTrainingPlansSQLRows(t, [][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte(`{"weeks":4}`), now, now, 1},
		}, nil)
		plans, totalCount, err := ScanTrainingPlans(rows)
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "plan-1", plans[0].ID)
		assert.Equal(t, "user-1", plans[0].UserID)
		assert.Equal(t, "strength", plans[0].Classification)
		assert.Equal(t, 4, plans[0].DurationWeeks)
		assert.Equal(t, []int{1, 3, 5}, plans[0].AvailableDays)
		assert.Equal(t, float64(4), plans[0].PlanData["weeks"])
		assert.Equal(t, 1, totalCount)
	})

	t.Run("returns nil plans for empty result", func(t *testing.T) {
		rows := newTrainingPlansSQLRows(t, nil, nil)
		plans, totalCount, err := ScanTrainingPlans(rows)
		require.NoError(t, err)
		assert.Nil(t, plans)
		assert.Equal(t, 0, totalCount)
	})

	t.Run("returns nil plans for empty plan data", func(t *testing.T) {
		now := time.Now()
		rows := newTrainingPlansSQLRows(t, [][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte{}, now, now, 3},
		}, nil)
		plans, totalCount, err := ScanTrainingPlans(rows)
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Nil(t, plans[0].PlanData)
		assert.Equal(t, 3, totalCount)
	})

	t.Run("returns error on scan failure", func(t *testing.T) {
		rows := newTrainingPlansSQLRows(t, [][]interface{}{
			{"plan-1", "user-1", "strength", "four", nil, []byte(`{}`), time.Now(), time.Now(), 1},
		}, nil)
		plans, totalCount, err := ScanTrainingPlans(rows)
		require.Error(t, err)
		assert.Nil(t, plans)
		assert.Equal(t, 0, totalCount)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		now := time.Now()
		rows := newTrainingPlansSQLRows(t, [][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte(`{"weeks":4}`), now, now, 1},
		}, assert.AnError)
		plans, totalCount, err := ScanTrainingPlans(rows)
		require.Error(t, err)
		assert.Nil(t, plans)
		assert.Equal(t, 0, totalCount)
	})
}

func TestScanTrainingPlansPGX(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		now := time.Now()
		rows := newMockPGXTrainingRows([][]interface{}{
			{"plan-1", "user-1", "strength", 4, []int{1, 3, 5}, []byte(`{"weeks":4}`), now, now, 1},
		})
		plans, totalCount, err := ScanTrainingPlansPGX(rows)
		require.NoError(t, err)
		require.Len(t, plans, 1)
		assert.Equal(t, "plan-1", plans[0].ID)
		assert.Equal(t, []int{1, 3, 5}, plans[0].AvailableDays)
		assert.Equal(t, float64(4), plans[0].PlanData["weeks"])
		assert.Equal(t, 1, totalCount)
	})

	t.Run("returns nil plans for empty result", func(t *testing.T) {
		plans, totalCount, err := ScanTrainingPlansPGX(newMockPGXTrainingRows(nil))
		require.NoError(t, err)
		assert.Nil(t, plans)
		assert.Equal(t, 0, totalCount)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		rows := newMockPGXTrainingRows(nil)
		rows.iterErr = assert.AnError
		plans, _, err := ScanTrainingPlansPGX(rows)
		require.Error(t, err)
		assert.Nil(t, plans)
	})
}

func TestScanTrainingPlanRows_UnmarshalError(t *testing.T) {
	rows := newRawTrainingPlanRows([]byte(`{"weeks":`))
	plans, totalCount, err := scanTrainingPlanRows(rows)
	require.Error(t, err)
	assert.Nil(t, plans)
	assert.Equal(t, 0, totalCount)
}

func TestScanAchievements(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		now := time.Now()
		rows := newAchievementsSQLRows(t, [][]interface{}{
			{"ach-1", "user-1", "milestone", "First Workout", "Complete first workout", now},
		}, nil)
		achievements, err := ScanAchievements(rows)
		require.NoError(t, err)
		require.Len(t, achievements, 1)
		assert.Equal(t, "ach-1", achievements[0].ID)
		assert.Equal(t, "user-1", achievements[0].UserID)
		assert.Equal(t, "milestone", achievements[0].Type)
		assert.Equal(t, "First Workout", achievements[0].Title)
		assert.Equal(t, "Complete first workout", achievements[0].Description)
		assert.Equal(t, now, achievements[0].EarnedAt)
	})

	t.Run("returns nil achievements for empty result", func(t *testing.T) {
		rows := newAchievementsSQLRows(t, nil, nil)
		achievements, err := ScanAchievements(rows)
		require.NoError(t, err)
		assert.Nil(t, achievements)
	})

	t.Run("returns error on scan failure", func(t *testing.T) {
		rows := newAchievementsSQLRows(t, [][]interface{}{
			{"ach-1", "user-1", "milestone", "First Workout", "Complete first workout", "not-a-time"},
		}, nil)
		achievements, err := ScanAchievements(rows)
		require.Error(t, err)
		assert.Nil(t, achievements)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		now := time.Now()
		rows := newAchievementsSQLRows(t, [][]interface{}{
			{"ach-1", "user-1", "milestone", "First Workout", "Complete first workout", now},
		}, assert.AnError)
		achievements, err := ScanAchievements(rows)
		require.Error(t, err)
		assert.Nil(t, achievements)
	})
}

func TestScanAchievementsPGX(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		now := time.Now()
		rows := newMockPGXAchievementRows([][]interface{}{
			{"ach-1", "user-1", "milestone", "First Workout", "Complete first workout", now},
		})
		achievements, err := ScanAchievementsPGX(rows)
		require.NoError(t, err)
		require.Len(t, achievements, 1)
		assert.Equal(t, "ach-1", achievements[0].ID)
		assert.Equal(t, "First Workout", achievements[0].Title)
		assert.Equal(t, now, achievements[0].EarnedAt)
	})

	t.Run("returns nil achievements for empty result", func(t *testing.T) {
		achievements, err := ScanAchievementsPGX(newMockPGXAchievementRows(nil))
		require.NoError(t, err)
		assert.Nil(t, achievements)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		rows := newMockPGXAchievementRows(nil)
		rows.iterErr = assert.AnError
		achievements, err := ScanAchievementsPGX(rows)
		require.Error(t, err)
		assert.Nil(t, achievements)
	})
}

type intSliceValueConverter struct{}

func (intSliceValueConverter) ConvertValue(v interface{}) (driver.Value, error) {
	if days, ok := v.([]int); ok {
		return days, nil
	}
	return driver.DefaultParameterConverter.ConvertValue(v)
}

func newTrainingPlansSQLRows(t *testing.T, values [][]interface{}, rowErr error) *sql.Rows {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(intSliceValueConverter{}))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	rows := mock.NewRows([]string{"id", "user_id", "classification", "duration_weeks", "available_days", "plan_data", "created_at", "updated_at", "total_count"})
	for _, value := range values {
		row := make([]driver.Value, len(value))
		for i, item := range value {
			row[i] = item
		}
		rows.AddRow(row...)
	}
	if rowErr != nil {
		rows.RowError(0, rowErr)
	}
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	result, err := db.Query("SELECT")
	require.NoError(t, err)
	return result
}

func newAchievementsSQLRows(t *testing.T, values [][]interface{}, rowErr error) *sql.Rows {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	rows := mock.NewRows([]string{"id", "user_id", "type", "title", "description", "earned_at"})
	for _, value := range values {
		row := make([]driver.Value, len(value))
		for i, item := range value {
			row[i] = item
		}
		rows.AddRow(row...)
	}
	if rowErr != nil {
		rows.RowError(0, rowErr)
	}
	mock.ExpectQuery("SELECT").WillReturnRows(rows)
	result, err := db.Query("SELECT")
	require.NoError(t, err)
	return result
}

type rawTrainingPlanRows struct {
	planData []byte
	consumed bool
}

func newRawTrainingPlanRows(planData []byte) *rawTrainingPlanRows {
	return &rawTrainingPlanRows{planData: planData}
}

func (m *rawTrainingPlanRows) Next() bool {
	if m.consumed {
		return false
	}
	m.consumed = true
	return true
}

func (m *rawTrainingPlanRows) Scan(dest ...interface{}) error {
	*dest[0].(*string) = "plan-1"
	*dest[1].(*string) = "user-1"
	*dest[2].(*string) = "strength"
	*dest[3].(*int) = 4
	*dest[4].(*[]int) = []int{1, 3, 5}
	*dest[5].(*[]byte) = m.planData
	*dest[6].(*time.Time) = time.Now()
	*dest[7].(*time.Time) = time.Now()
	*dest[8].(*int) = 1
	return nil
}

func (m *rawTrainingPlanRows) Err() error {
	return nil
}

type mockPGXTrainingRows struct {
	values  [][]interface{}
	idx     int
	scanErr error
	iterErr error
}

func newMockPGXTrainingRows(values [][]interface{}) *mockPGXTrainingRows {
	return &mockPGXTrainingRows{values: values}
}

func (m *mockPGXTrainingRows) Next() bool {
	if m.idx < len(m.values) {
		m.idx++
		return true
	}
	return false
}

func (m *mockPGXTrainingRows) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	value := m.values[m.idx-1]
	for i, d := range dest {
		if err := assignScanValue(d, value[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockPGXTrainingRows) Err() error {
	return m.iterErr
}

func (m *mockPGXTrainingRows) Close() {}

func (m *mockPGXTrainingRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *mockPGXTrainingRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *mockPGXTrainingRows) Values() ([]interface{}, error) {
	return nil, assert.AnError
}

func (m *mockPGXTrainingRows) RawValues() [][]byte {
	return nil
}

func (m *mockPGXTrainingRows) Conn() *pgx.Conn {
	return nil
}

func (m *mockPGXTrainingRows) TypeMap() *pgtype.Map {
	return nil
}

type mockPGXAchievementRows struct {
	values  [][]interface{}
	idx     int
	scanErr error
	iterErr error
}

func newMockPGXAchievementRows(values [][]interface{}) *mockPGXAchievementRows {
	return &mockPGXAchievementRows{values: values}
}

func (m *mockPGXAchievementRows) Next() bool {
	if m.idx < len(m.values) {
		m.idx++
		return true
	}
	return false
}

func (m *mockPGXAchievementRows) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	value := m.values[m.idx-1]
	for i, d := range dest {
		if err := assignScanValue(d, value[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockPGXAchievementRows) Err() error {
	return m.iterErr
}

func (m *mockPGXAchievementRows) Close() {}

func (m *mockPGXAchievementRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (m *mockPGXAchievementRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}

func (m *mockPGXAchievementRows) Values() ([]interface{}, error) {
	return nil, assert.AnError
}

func (m *mockPGXAchievementRows) RawValues() [][]byte {
	return nil
}

func (m *mockPGXAchievementRows) Conn() *pgx.Conn {
	return nil
}

func (m *mockPGXAchievementRows) TypeMap() *pgtype.Map {
	return nil
}

func assignScanValue(dest interface{}, value interface{}) error {
	switch ptr := dest.(type) {
	case *string:
		s, ok := value.(string)
		if !ok {
			return assert.AnError
		}
		*ptr = s
	case *int:
		n, ok := value.(int)
		if !ok {
			return assert.AnError
		}
		*ptr = n
	case *[]int:
		n, ok := value.([]int)
		if !ok {
			return assert.AnError
		}
		*ptr = n
	case *[]byte:
		b, ok := value.([]byte)
		if !ok {
			return assert.AnError
		}
		*ptr = b
	case *time.Time:
		ts, ok := value.(time.Time)
		if !ok {
			return assert.AnError
		}
		*ptr = ts
	default:
		return assert.AnError
	}
	return nil
}
