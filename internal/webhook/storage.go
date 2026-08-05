// Package webhook provides Open Wearables webhook handling.
package webhook

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/MAMUER/project/internal/sanitize"
)

const (
	insertBiometricMetricQuery = `
		INSERT INTO biometric_data (
			user_id, metric_type, value, timestamp, device_type, source, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, metric_type, timestamp, source)
		DO UPDATE SET
			value = EXCLUDED.value,
			device_type = EXCLUDED.device_type
	`

	insertWebhookNonceQuery = `
		INSERT INTO webhook_nonces (user_id, nonce, created_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, nonce) DO NOTHING
	`

	checkWebhookNonceQuery = `
		SELECT created_at FROM webhook_nonces
		WHERE user_id = $1 AND nonce = $2
	`

	getSourcesQuery = `
		SELECT DISTINCT source, MAX(created_at)
		FROM biometric_data
		WHERE user_id = $1 AND device_type = 'open_wearables'
		GROUP BY source
	`

	deleteBySourceQuery = `
		DELETE FROM biometric_data
		WHERE user_id = $1 AND source = $2 AND device_type = 'open_wearables'
	`
)

// Storage handles persistence of webhook metrics
type Storage struct {
	db  DB
	log *zap.Logger
}

// DB is an interface for database operations
type DB interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	GetSources(ctx context.Context, userID string) ([]SourceInfo, error)
	DeleteBySource(ctx context.Context, userID, source string) (int64, error)
}

// RowScanner abstracts sql.Row Scan
type RowScanner interface {
	Scan(dest ...interface{}) error
}

// Tx is an interface for database transactions
type Tx interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner
	Commit() error
	Rollback() error
}

// Result represents the result of a database operation
type Result interface {
	RowsAffected() (int64, error)
}

// NewStorage creates a new webhook storage
func NewStorage(db DB, log *zap.Logger) *Storage {
	return &Storage{
		db:  db,
		log: log.Named("webhook.storage"),
	}
}

// SaveMetrics saves multiple metrics in a single transaction
func (s *Storage) SaveMetrics(ctx context.Context, payload *OpenWearablesWebhookPayload) error {
	if len(payload.Metrics) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, metric := range payload.Metrics {
		if err := s.saveMetric(ctx, tx, payload, metric); err != nil {
			return fmt.Errorf("save metric %s for user %s: %w",
				sanitize.LogString(string(metric.Type)), sanitize.LogString(payload.UserID), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	s.log.Info("metrics saved",
		zap.String("user_id", sanitize.LogString(payload.UserID)),
		zap.String("source", sanitize.LogString(string(payload.Source))),
		zap.Int("count", len(payload.Metrics)),
	)

	return nil
}

func (s *Storage) saveMetric(ctx context.Context, tx Tx, payload *OpenWearablesWebhookPayload, metric OpenWearablesMetric) error {
	ts := metric.Timestamp
	if ts.IsZero() {
		ts = payload.Timestamp
	}

	_, err := tx.ExecContext(ctx, insertBiometricMetricQuery,
		payload.UserID,
		metric.Type,
		metric.Value,
		ts,
		string(payload.Source),
		string(payload.Source),
	)
	if err != nil {
		return fmt.Errorf("insert metric: %w", err)
	}

	return nil
}

// SourceInfo represents a connected source
type SourceInfo struct {
	Source      string    `json:"source"`
	SourceName  string    `json:"source_name"`
	ConnectedAt time.Time `json:"connected_at"`
}

// GetSources returns distinct Open Wearables sources for a user
func (s *Storage) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return s.db.GetSources(ctx, userID)
}

// DeleteBySource removes biometric data for a user and source
func (s *Storage) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return s.db.DeleteBySource(ctx, userID, source)
}

// CheckAndSaveNonce checks if a nonce has already been used and saves it
func (s *Storage) CheckAndSaveNonce(ctx context.Context, userID, nonce string, timestamp time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var createdAt time.Time
	err = tx.QueryRowContext(ctx, checkWebhookNonceQuery, userID, nonce).Scan(&createdAt)
	if err == nil {
		return fmt.Errorf("nonce already used at %s", createdAt.Format(time.RFC3339))
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("check nonce: %w", err)
	}

	_, err = tx.ExecContext(ctx, insertWebhookNonceQuery, userID, nonce)
	if err != nil {
		return fmt.Errorf("insert nonce: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit nonce transaction: %w", err)
	}

	return nil
}

// SQLDBAdapter adapts *sql.DB to webhook.DB
type SQLDBAdapter struct {
	db *sql.DB
}

func NewSQLDBAdapter(db *sql.DB) *SQLDBAdapter {
	return &SQLDBAdapter{db: db}
}

func (a *SQLDBAdapter) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := a.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &SQLTxAdapter{tx: tx}, nil // NOSONAR godre:S8168 - transaction lifecycle is managed by the caller, which uses defer tx.Rollback()
}

func (a *SQLDBAdapter) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	rows, err := a.db.QueryContext(ctx, getSourcesQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var sources []SourceInfo
	for rows.Next() {
		var source string
		var connectedAt time.Time
		if err := rows.Scan(&source, &connectedAt); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		sources = append(sources, SourceInfo{
			Source:      source,
			SourceName:  source,
			ConnectedAt: connectedAt,
		})
	}

	return sources, rows.Err()
}

func (a *SQLDBAdapter) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	result, err := a.db.ExecContext(ctx, deleteBySourceQuery, userID, source)
	if err != nil {
		return 0, fmt.Errorf("delete by source: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}

	return count, nil
}

// SQLTxAdapter adapts *sql.Tx to webhook.Tx
type SQLTxAdapter struct {
	tx *sql.Tx
}

func (a *SQLTxAdapter) ExecContext(ctx context.Context, query string, args ...interface{}) (Result, error) {
	return a.tx.ExecContext(ctx, query, args...)
}

func (a *SQLTxAdapter) QueryRowContext(ctx context.Context, query string, args ...interface{}) RowScanner {
	return a.tx.QueryRowContext(ctx, query, args...)
}

func (a *SQLTxAdapter) Commit() error {
	return a.tx.Commit()
}

func (a *SQLTxAdapter) Rollback() error {
	return a.tx.Rollback()
}

// SQLResultAdapter adapts sql.Result to webhook.Result
type SQLResultAdapter struct {
	result sql.Result
}

func NewSQLResultAdapter(result sql.Result) *SQLResultAdapter {
	return &SQLResultAdapter{result: result}
}

func (a *SQLResultAdapter) RowsAffected() (int64, error) {
	return a.result.RowsAffected()
}
