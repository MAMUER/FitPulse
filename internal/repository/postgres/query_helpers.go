package postgres

import (
	"database/sql"
	"time"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

const (
	selectPrefix = "SELECT "
	fromPrefix   = " FROM "
	wherePrefix  = " WHERE "
)

// BuildDateRangeQuery builds a SQL query with optional date range filtering.
// table: the table name (already validated/constant)
// columns: the SELECT columns
// userIDColumn: the user id column name, typically "user_id"
// limitClause: the LIMIT/OFFSET clause, e.g. "ORDER BY recorded_at DESC LIMIT $N"
func BuildDateRangeQuery(table, columns, userIDColumn, userID string, from, to *time.Time, limit int) (string, []interface{}) {
	var query string
	var args []interface{}

	switch {
	case from != nil && to != nil:
		query = selectPrefix + columns + fromPrefix + table + wherePrefix + userIDColumn + " = $1 AND recorded_at >= $2 AND recorded_at <= $3 ORDER BY recorded_at DESC LIMIT $4"
		args = []interface{}{userID, *from, *to, limit}
	case from != nil:
		query = selectPrefix + columns + fromPrefix + table + wherePrefix + userIDColumn + " = $1 AND recorded_at >= $2 ORDER BY recorded_at DESC LIMIT $3"
		args = []interface{}{userID, *from, limit}
	case to != nil:
		query = selectPrefix + columns + fromPrefix + table + wherePrefix + userIDColumn + " = $1 AND recorded_at <= $2 ORDER BY recorded_at DESC LIMIT $3"
		args = []interface{}{userID, *to, limit}
	default:
		query = selectPrefix + columns + fromPrefix + table + wherePrefix + userIDColumn + " = $1 ORDER BY recorded_at DESC LIMIT $2"
		args = []interface{}{userID, limit}
	}

	return query, args
}

// ScanRows scans rows into a slice using the provided scan function.
// scanFunc should scan the current row into an entity and return it.
func ScanRows[T any](rows *sql.Rows, scanFunc func(*sql.Rows) (T, error)) ([]T, error) {
	var results []T
	for rows.Next() {
		item, err := scanFunc(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// ScanDevices scans rows into devices.
func ScanDevices(rows *sql.Rows) ([]*entity.Device, error) {
	var devices []*entity.Device
	for rows.Next() {
		d := &entity.Device{}
		if err := rows.Scan(&d.ID, &d.UserID, &d.DeviceType, &d.DeviceName, &d.IsConnected, &d.LastSync); err != nil {
			return nil, apperrors.Internal("failed to scan device", err)
		}
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate devices", err)
	}
	return devices, nil
}
