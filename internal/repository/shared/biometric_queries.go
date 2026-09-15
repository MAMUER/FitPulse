// Package shared contains reusable biometric queries.
package shared

import (
	"fmt"

	"github.com/MAMUER/project/internal/apperrors"
	"github.com/MAMUER/project/internal/domain/entity"
)

const (
	BiometricInsertCreateQuery = `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, metric_type, timestamp, source) DO UPDATE SET id = biometric_data.id
		RETURNING id
	`

	BiometricBatchCreateQuery = `
		INSERT INTO biometric_data (id, user_id, metric_type, value, timestamp, device_type, source, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (user_id, metric_type, timestamp, source) DO NOTHING
	`

	BiometricGetByUserIDBaseQuery = `
		SELECT id, user_id, metric_type, value, timestamp, device_type, source, created_at
		FROM biometric_data
		WHERE user_id = $1
	`

	BiometricGetLatestQuery = `
		SELECT id, user_id, metric_type, value, timestamp, device_type, source, created_at
		FROM biometric_data
		WHERE user_id = $1 AND metric_type = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	BiometricUpdateQuery = `
		UPDATE biometric_data
		SET value = $1, timestamp = $2, device_type = $3
		WHERE id = $4
		RETURNING id, user_id, metric_type, value, timestamp, device_type, source, created_at
	`

	BiometricDeleteQuery = `DELETE FROM biometric_data WHERE id = $1`
)

type biometricRowScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}

func ScanBiometricRows(rows biometricRowScanner) ([]*entity.BiometricRecord, error) {
	var records []*entity.BiometricRecord
	for rows.Next() {
		record := &entity.BiometricRecord{}
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.MetricType, &record.Value,
			&record.Timestamp, &record.DeviceType, &record.Source, &record.CreatedAt,
		); err != nil {
			return nil, apperrors.Internal("failed to scan biometric record", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, apperrors.Internal("failed to iterate biometric records", err)
	}
	return records, nil
}

func argPlaceholder(n int) string {
	if n == 0 {
		return "1"
	}
	return fmt.Sprintf("$%d", n)
}

func BuildBiometricQuery(userID, metricType string, limit, offset int) (string, []interface{}) {
	args := []interface{}{userID}
	argCount := 1
	query := BiometricGetByUserIDBaseQuery

	if metricType != "" {
		argCount++
		query += " AND metric_type = " + argPlaceholder(argCount)
		args = append(args, metricType)
	}
	argCount++
	query += " ORDER BY timestamp DESC LIMIT " + argPlaceholder(argCount)
	args = append(args, limit)
	if offset > 0 {
		argCount++
		query += " OFFSET " + argPlaceholder(argCount)
		args = append(args, offset)
	}
	return query, args
}
