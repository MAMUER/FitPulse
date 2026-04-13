package utils

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockCloser allows controlling Close behavior in tests
type mockCloser struct {
	closeErr error
	closed   bool
}

func (m *mockCloser) Close() error {
	m.closed = true
	return m.closeErr
}

func TestCloseIgnoreWithNilCloser(t *testing.T) {
	CloseIgnore(nil) // should not panic
}

func TestCloseIgnoreWithValidCloser(t *testing.T) {
	mc := &mockCloser{}
	CloseIgnore(mc)
	assert.True(t, mc.closed)
}

func TestCloseIgnoreIgnoresError(t *testing.T) {
	mc := &mockCloser{closeErr: errors.New("close failed")}
	CloseIgnore(mc) // should not panic or fail
	assert.True(t, mc.closed)
}

func TestCloseDBWithNilDB(t *testing.T) {
	CloseDB(nil, nil) // should not panic
}

func TestCloseDBSuccess(t *testing.T) {
	var logged string
	logFn := func(msg string, args ...interface{}) {
		logged = msg
	}

	// Create a real sql.DB that's already "closed" — use nil to test nil check
	// We can't easily create a real *sql.DB without a real DB, so test nil
	CloseDB(nil, logFn)
	assert.Empty(t, logged)
}

func TestCloseRowsWithNilRows(t *testing.T) {
	CloseRows(nil, nil) // should not panic
}

func TestCloseSafeWithNilCloser(t *testing.T) {
	CloseSafe(nil, nil) // should not panic
}

func TestCloseSafeSuccess(t *testing.T) {
	var logged string
	logFn := func(msg string, args ...interface{}) {
		logged = msg
	}

	mc := &mockCloser{}
	CloseSafe(mc, logFn)
	assert.True(t, mc.closed)
	assert.Empty(t, logged)
}

func TestCloseSafeWithError(t *testing.T) {
	var logged string
	logFn := func(msg string, args ...interface{}) {
		logged = msg
	}

	mc := &mockCloser{closeErr: errors.New("close failed")}
	CloseSafe(mc, logFn)
	assert.True(t, mc.closed)
	assert.Contains(t, logged, "Failed to close resource")
}

func TestCloseWithErrorNil(t *testing.T) {
	err := CloseWithError(nil)
	assert.NoError(t, err)
}

func TestCloseWithErrorSuccess(t *testing.T) {
	mc := &mockCloser{}
	err := CloseWithError(mc)
	assert.NoError(t, err)
	assert.True(t, mc.closed)
}

func TestCloseWithErrorReturnsError(t *testing.T) {
	expectedErr := errors.New("close failed")
	mc := &mockCloser{closeErr: expectedErr}
	err := CloseWithError(mc)
	assert.Equal(t, expectedErr, err)
}

func TestMultiCloserEmpty(t *testing.T) {
	mc := MultiCloser{}
	err := mc.Close()
	assert.NoError(t, err)
}

func TestMultiCloserSuccess(t *testing.T) {
	c1 := &mockCloser{}
	c2 := &mockCloser{}
	mc := MultiCloser{c1, c2}
	err := mc.Close()
	assert.NoError(t, err)
	assert.True(t, c1.closed)
	assert.True(t, c2.closed)
}

func TestMultiCloserWithNil(t *testing.T) {
	c1 := &mockCloser{}
	mc := MultiCloser{nil, c1, nil}
	err := mc.Close()
	assert.NoError(t, err)
	assert.True(t, c1.closed)
}

func TestMultiCloserWithError(t *testing.T) {
	c1 := &mockCloser{}
	c2 := &mockCloser{closeErr: errors.New("fail")}
	c3 := &mockCloser{}
	mc := MultiCloser{c1, c2, c3}
	err := mc.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multi-close errors")
	assert.True(t, c1.closed)
	assert.True(t, c2.closed)
	assert.True(t, c3.closed)
}

func TestMultiCloserMultipleErrors(t *testing.T) {
	c1 := &mockCloser{closeErr: errors.New("err1")}
	c2 := &mockCloser{closeErr: errors.New("err2")}
	mc := MultiCloser{c1, c2}
	err := mc.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "err1")
	assert.Contains(t, err.Error(), "err2")
}

func TestMultiCloserAdd(t *testing.T) {
	mc := &MultiCloser{}
	c1 := &mockCloser{}
	c2 := &mockCloser{}
	mc.Add(c1)
	mc.Add(c2)
	assert.Len(t, *mc, 2)
	_ = mc.Close()
	assert.True(t, c1.closed)
	assert.True(t, c2.closed)
}

func TestMultiCloserAddNil(t *testing.T) {
	mc := &MultiCloser{}
	mc.Add(nil) // should not add
	assert.Len(t, *mc, 0)
}

func TestMultiCloserImplementsCloser(t *testing.T) {
	mc := MultiCloser{}
	var _ io.Closer = mc
}

// nopCloser is a test double for io.Closer
type nopCloser struct{}

func (nopCloser) Close() error { return nil }

func TestCloseIgnoreWithNopCloser(t *testing.T) {
	var nc nopCloser
	CloseIgnore(&nc) // should not panic
}

func TestCloseSafeWithStringLikeCloser(t *testing.T) {
	// Test with a custom io.Closer implementation
	var logged string
	logFn := func(msg string, args ...interface{}) {
		logged = msg
	}
	CloseSafe(nopCloser{}, logFn)
	assert.Empty(t, logged)
}

// stringsCloser always succeeds
type stringsCloser struct{}

func (stringsCloser) Close() error { return nil }

// errorCloser always fails
type errorCloser struct{}

func (errorCloser) Close() error { return errors.New("always fails") }

func TestMultiCloserImplementsClose(t *testing.T) {
	mc := make(MultiCloser, 0, 1)
	mc = append(mc, stringsCloser{})
	err := mc.Close()
	assert.NoError(t, err)
}

func TestMultiCloserAllClosedOnError(t *testing.T) {
	mc := MultiCloser{errorCloser{}, stringsCloser{}, errorCloser{}}
	err := mc.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multi-close errors")
}
