package webhook

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockRowsForSources struct {
	sources []SourceInfo
	idx     int
}

func (r *mockRowsForSources) Next() bool {
	r.idx++
	return r.idx <= len(r.sources)
}

func (r *mockRowsForSources) Scan(dest ...interface{}) error {
	if r.idx-1 < len(r.sources) {
		src := r.sources[r.idx-1]
		if s, ok := dest[0].(*string); ok {
			*s = src.Source
		}
		if t, ok := dest[1].(*time.Time); ok {
			*t = src.ConnectedAt
		}
	}
	return nil
}

func (r *mockRowsForSources) Close() error {
	return nil
}

func (r *mockRowsForSources) Err() error {
	return nil
}

type mockDBForSources struct {
	sources []SourceInfo
}

func (m *mockDBForSources) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	return &mockTx{}, nil
}

func (m *mockDBForSources) GetSources(ctx context.Context, userID string) ([]SourceInfo, error) {
	return m.sources, nil
}

func (m *mockDBForSources) DeleteBySource(ctx context.Context, userID, source string) (int64, error) {
	return 1, nil
}

func TestStorageGetSources(t *testing.T) {
	t.Run("returns sources", func(t *testing.T) {
		now := time.Now()
		db := &mockDBForSources{
			sources: []SourceInfo{
				{Source: "open_wearables", SourceName: "Open Wearables", ConnectedAt: now},
			},
		}
		store := NewStorage(db, zap.NewNop())
		sources, err := store.GetSources(context.Background(), "user-1")
		assert.NoError(t, err)
		assert.Len(t, sources, 1)
		assert.Equal(t, "open_wearables", sources[0].Source)
	})
}

func TestStorageDeleteBySource(t *testing.T) {
	t.Run("deletes records", func(t *testing.T) {
		db := &mockDBForSources{}
		store := NewStorage(db, zap.NewNop())
		count, err := store.DeleteBySource(context.Background(), "user-1", "open_wearables")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}
