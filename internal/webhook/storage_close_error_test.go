package webhook

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSQLDBAdapterGetSourcesReportsCloseErrors exercises the deferred rows.Close
// error path in GetSources. The row set reports a further result set so
// database/sql does not close it on EOF, which keeps the close error available
// to the deferred call.
func TestSQLDBAdapterGetSourcesReportsCloseErrors(t *testing.T) {
	db := sql.OpenDB(closeErrorConnector{closeErr: errors.New("close failed")})
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	adapter := NewSQLDBAdapter(db)

	sources, err := adapter.GetSources(context.Background(), "user-1")

	require.Error(t, err)
	assert.Nil(t, sources)
	assert.Contains(t, err.Error(), "close sources rows")
	assert.Contains(t, err.Error(), "close failed")
}

type closeErrorConnector struct {
	closeErr error
}

func (c closeErrorConnector) Connect(context.Context) (driver.Conn, error) {
	return &closeErrorConn{closeErr: c.closeErr}, nil
}

func (closeErrorConnector) Driver() driver.Driver {
	return closeErrorDriver{}
}

type closeErrorDriver struct{}

func (closeErrorDriver) Open(string) (driver.Conn, error) {
	return &closeErrorConn{}, nil
}

type closeErrorConn struct {
	closeErr error
}

func (*closeErrorConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare not supported")
}

func (*closeErrorConn) Close() error {
	return nil
}

func (*closeErrorConn) Begin() (driver.Tx, error) {
	return nil, errors.New("transactions not supported")
}

func (c *closeErrorConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &closeErrorRows{closeErr: c.closeErr}, nil
}

type closeErrorRows struct {
	closeErr error
}

func (*closeErrorRows) Columns() []string {
	return []string{"source", "connected_at"}
}

func (r *closeErrorRows) Close() error {
	return r.closeErr
}

func (*closeErrorRows) Next([]driver.Value) error {
	return io.EOF
}

func (*closeErrorRows) HasNextResultSet() bool {
	return true
}

func (*closeErrorRows) NextResultSet() error {
	return io.EOF
}
