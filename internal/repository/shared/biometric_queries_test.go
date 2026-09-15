package shared

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/domain/entity"
)

func TestBuildBiometricQuery(t *testing.T) {
	t.Run("userID only", func(t *testing.T) {
		query, args := BuildBiometricQuery("user-1", "", 10, 0)
		assert.Contains(t, query, "WHERE user_id = $1")
		assert.Contains(t, query, "LIMIT $2")
		assert.NotContains(t, query, "OFFSET")
		assert.Equal(t, []interface{}{"user-1", 10}, args)
	})

	t.Run("userID with offset", func(t *testing.T) {
		query, args := BuildBiometricQuery("user-1", "", 10, 20)
		assert.Contains(t, query, "WHERE user_id = $1")
		assert.Contains(t, query, "LIMIT $2")
		assert.Contains(t, query, "OFFSET $3")
		assert.Equal(t, []interface{}{"user-1", 10, 20}, args)
	})

	t.Run("userID with metricType", func(t *testing.T) {
		query, args := BuildBiometricQuery("user-1", "heart_rate", 10, 0)
		assert.Contains(t, query, "WHERE user_id = $1")
		assert.Contains(t, query, "AND metric_type = $2")
		assert.Contains(t, query, "LIMIT $3")
		assert.Equal(t, []interface{}{"user-1", "heart_rate", 10}, args)
	})

	t.Run("userID with metricType and offset", func(t *testing.T) {
		query, args := BuildBiometricQuery("user-1", "heart_rate", 10, 20)
		assert.Contains(t, query, "WHERE user_id = $1")
		assert.Contains(t, query, "AND metric_type = $2")
		assert.Contains(t, query, "LIMIT $3")
		assert.Contains(t, query, "OFFSET $4")
		assert.Equal(t, []interface{}{"user-1", "heart_rate", 10, 20}, args)
	})
}

func TestScanBiometricRows(t *testing.T) {
	t.Run("scans rows successfully", func(t *testing.T) {
		now := time.Now()
		rows := newMockBiometricRows([]*entity.BiometricRecord{
			{ID: "b1", UserID: "u1", MetricType: "heart_rate", Value: 70, Timestamp: now, DeviceType: "watch", Source: "test", CreatedAt: now},
		})
		records, err := ScanBiometricRows(rows)
		require.NoError(t, err)
		require.Len(t, records, 1)
		assert.Equal(t, "b1", records[0].ID)
		assert.Equal(t, "u1", records[0].UserID)
		assert.Equal(t, "heart_rate", records[0].MetricType)
		assert.Equal(t, 70.0, records[0].Value)
	})

	t.Run("returns error on scan failure", func(t *testing.T) {
		rows := newMockBiometricRows([]*entity.BiometricRecord{{}})
		rows.scanErr = assert.AnError
		records, err := ScanBiometricRows(rows)
		assert.Nil(t, records)
		assert.Error(t, err)
	})

	t.Run("returns error on rows iteration failure", func(t *testing.T) {
		rows := newMockBiometricRows(nil)
		rows.iterErr = assert.AnError
		records, err := ScanBiometricRows(rows)
		assert.Nil(t, records)
		assert.Error(t, err)
	})
}

type mockBiometricRows struct {
	records   []*entity.BiometricRecord
	idx       int
	scanErr   error
	iterErr   error
}

func newMockBiometricRows(records []*entity.BiometricRecord) *mockBiometricRows {
	return &mockBiometricRows{records: records}
}

func (m *mockBiometricRows) Next() bool {
	if m.idx < len(m.records) {
		m.idx++
		return true
	}
	return false
}

func (m *mockBiometricRows) Scan(dest ...interface{}) error {
	if m.scanErr != nil {
		return m.scanErr
	}
	if m.idx-1 < len(m.records) {
		r := m.records[m.idx-1]
		id := dest[0].(*string)
		userID := dest[1].(*string)
		metricType := dest[2].(*string)
		value := dest[3].(*float64)
		timestamp := dest[4].(*time.Time)
		deviceType := dest[5].(*string)
		source := dest[6].(*string)
		createdAt := dest[7].(*time.Time)
		*id = r.ID
		*userID = r.UserID
		*metricType = r.MetricType
		*value = r.Value
		*timestamp = r.Timestamp
		*deviceType = r.DeviceType
		*source = r.Source
		*createdAt = r.CreatedAt
	}
	return nil
}

func (m *mockBiometricRows) Err() error {
	return m.iterErr
}

func (m *mockBiometricRows) Close() {}
