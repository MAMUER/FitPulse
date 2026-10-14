package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MAMUER/project/internal/logger"
)

func TestQueueMetricsSetNilReceiver(t *testing.T) {
	var m *QueueMetrics
	assert.NotPanics(t, func() { m.Set(10) })
}

func TestExportQueueDepth(t *testing.T) {
	assert.NotPanics(t, func() {
		ExportQueueDepth("notifications", "high", 5)
	})
}

func TestRegisterQueueMetrics_SameInstance(t *testing.T) {
	m1 := registerQueueMetrics("registration-test", "default")
	m2 := registerQueueMetrics("registration-test", "default")
	require.NotNil(t, m1)
	assert.Same(t, m1, m2)

	other := registerQueueMetrics("registration-test", "urgent")
	assert.NotSame(t, m1, other)
}

func TestWithPublisherPriority(t *testing.T) {
	opts := &publisherOptions{}
	WithPublisherPriority("urgent")(opts)
	assert.Equal(t, "urgent", opts.priority)
}

func TestWithConsumerPriority(t *testing.T) {
	opts := &consumerOptions{}
	WithConsumerPriority("low")(opts)
	assert.Equal(t, "low", opts.priority)
}

func TestEnsureLogger(t *testing.T) {
	created := ensureLogger(nil)
	require.NotNil(t, created)
	assert.Equal(t, "queue", created.Service())

	own := logger.New("custom")
	assert.Same(t, own, ensureLogger(own))
}
