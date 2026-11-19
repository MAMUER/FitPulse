package email

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTask_JSONRoundTrip(t *testing.T) {
	task := Task{
		Type:    TypeVerification,
		To:      "user@example.com",
		Token:   "tok123",
		BaseURL: "https://fitpulse.example",
	}
	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded Task
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, task, decoded)
}

func TestQueueName(t *testing.T) {
	assert.Equal(t, "emails", QueueName)
}

func TestTypeVerification(t *testing.T) {
	assert.Equal(t, "verification", TypeVerification)
}
