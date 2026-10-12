package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type recordingLogger struct {
	events []Event
}

func (r *recordingLogger) Log(event Event) {
	r.events = append(r.events, event)
}

func TestFieldMaskMarshalJSON(t *testing.T) {
	t.Run("nil mask marshals to empty array", func(t *testing.T) {
		var mask FieldMask
		data, err := mask.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})

	t.Run("populated mask marshals to array", func(t *testing.T) {
		mask := FieldMask{"email", "phone"}
		data, err := mask.MarshalJSON()
		require.NoError(t, err)
		assert.JSONEq(t, `["email","phone"]`, string(data))
	})

	t.Run("empty mask marshals to empty array", func(t *testing.T) {
		mask := FieldMask{}
		data, err := mask.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, "[]", string(data))
	})
}

func TestEventJSONSerialization(t *testing.T) {
	t.Run("full event keeps all fields", func(t *testing.T) {
		event := NewEvent("user-service", EventTypePIIUpdate, "update_profile", "users/42").
			WithUser("42").
			WithActor("admin-1").
			WithFields("email", "phone").
			WithCorrelationID("corr-1").
			WithMetadata("ip", "127.0.0.1")

		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &decoded))

		assert.Equal(t, "user-service", decoded["service"])
		assert.Equal(t, "pii_update", decoded["event_type"])
		assert.Equal(t, "42", decoded["user_id"])
		assert.Equal(t, "admin-1", decoded["actor_id"])
		assert.Equal(t, "update_profile", decoded["action"])
		assert.Equal(t, "users/42", decoded["resource"])
		assert.Equal(t, []interface{}{"email", "phone"}, decoded["fields"])
		assert.Equal(t, true, decoded["success"])
		assert.Equal(t, "corr-1", decoded["correlation_id"])
		assert.Equal(t, map[string]interface{}{"ip": "127.0.0.1"}, decoded["metadata"])
		assert.NotEmpty(t, decoded["timestamp"])
	})

	t.Run("empty event omits optional fields", func(t *testing.T) {
		var event Event
		data, err := json.Marshal(event)
		require.NoError(t, err)

		raw := string(data)
		assert.NotContains(t, raw, "user_id")
		assert.NotContains(t, raw, "actor_id")
		assert.NotContains(t, raw, "error")
		assert.NotContains(t, raw, "fields")
		assert.NotContains(t, raw, "metadata")
		assert.NotContains(t, raw, "correlation_id")
		assert.Contains(t, raw, `"service":""`)
		assert.Contains(t, raw, `"success":false`)
	})

	t.Run("failed event keeps error message", func(t *testing.T) {
		event := NewEvent("user-service", EventTypeAuthFailure, "login", "auth").WithError(errors.New("bad password"))
		data, err := json.Marshal(event)
		require.NoError(t, err)

		var decoded map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, "bad password", decoded["error"])
		assert.Equal(t, false, decoded["success"])
	})
}

func TestNewEvent(t *testing.T) {
	event := NewEvent("gateway", EventTypeAdminAction, "delete_user", "users/7")

	assert.Equal(t, "gateway", event.Service)
	assert.Equal(t, EventTypeAdminAction, event.EventType)
	assert.Equal(t, "delete_user", event.Action)
	assert.Equal(t, "users/7", event.Resource)
	assert.True(t, event.Success)
	assert.Empty(t, event.UserID)
	assert.Empty(t, event.ActorID)
	assert.Empty(t, event.Fields)
	assert.Empty(t, event.Error)
	assert.Empty(t, event.CorrelationID)
	assert.NotNil(t, event.Metadata)
	assert.Empty(t, event.Metadata)
	assert.Equal(t, time.UTC, event.Timestamp.Location())
}

func TestEventWithUser(t *testing.T) {
	event := NewEvent("svc", EventTypePIIAccess, "read", "users/1")
	updated := event.WithUser("user-9")

	assert.Equal(t, "user-9", updated.UserID)
	assert.Empty(t, event.UserID)
}

func TestEventWithActor(t *testing.T) {
	event := NewEvent("svc", EventTypeAdminAction, "ban", "users/1")
	updated := event.WithActor("admin-3")

	assert.Equal(t, "admin-3", updated.ActorID)
	assert.Empty(t, event.ActorID)
}

func TestEventWithFields(t *testing.T) {
	t.Run("appends to nil mask", func(t *testing.T) {
		event := NewEvent("svc", EventTypePIIUpdate, "update", "users/1")
		updated := event.WithFields("email").WithFields("phone", "birthdate")

		assert.Equal(t, FieldMask{"email", "phone", "birthdate"}, updated.Fields)
		assert.Empty(t, event.Fields)
	})

	t.Run("appends to existing mask", func(t *testing.T) {
		event := Event{Fields: FieldMask{"email"}}
		updated := event.WithFields("phone")

		assert.Equal(t, FieldMask{"email", "phone"}, updated.Fields)
	})
}

func TestEventWithCorrelationID(t *testing.T) {
	event := NewEvent("svc", EventTypePIIExport, "export", "users/1")
	updated := event.WithCorrelationID("trace-42")

	assert.Equal(t, "trace-42", updated.CorrelationID)
	assert.Empty(t, event.CorrelationID)
}

func TestEventWithError(t *testing.T) {
	t.Run("nil error keeps event successful", func(t *testing.T) {
		event := NewEvent("svc", EventTypeAuthSuccess, "login", "auth")
		updated := event.WithError(nil)

		assert.True(t, updated.Success)
		assert.Empty(t, updated.Error)
	})

	t.Run("non-nil error marks failure", func(t *testing.T) {
		event := NewEvent("svc", EventTypeAuthFailure, "login", "auth")
		updated := event.WithError(errors.New("invalid credentials"))

		assert.False(t, updated.Success)
		assert.Equal(t, "invalid credentials", updated.Error)
		assert.True(t, event.Success)
	})

	t.Run("wrapped error keeps message", func(t *testing.T) {
		base := errors.New("connection refused")
		updated := NewEvent("svc", EventTypeAuthFailure, "login", "auth").WithError(base)

		assert.Equal(t, "connection refused", updated.Error)
		assert.False(t, updated.Success)
	})
}

func TestEventWithMetadata(t *testing.T) {
	t.Run("creates map when nil", func(t *testing.T) {
		event := Event{}
		updated := event.WithMetadata("attempts", 3)

		require.NotNil(t, updated.Metadata)
		assert.Equal(t, 3, updated.Metadata["attempts"])
		assert.Nil(t, event.Metadata)
	})

	t.Run("adds to existing map and overwrites keys", func(t *testing.T) {
		event := NewEvent("svc", EventTypePIIAccess, "read", "users/1")
		updated := event.WithMetadata("ip", "10.0.0.1").WithMetadata("ip", "10.0.0.2")

		assert.Equal(t, "10.0.0.2", updated.Metadata["ip"])
		assert.Len(t, updated.Metadata, 1)
	})
}

func TestEventValidate(t *testing.T) {
	valid := NewEvent("svc", EventTypePIIAccess, "read", "users/1")

	require.NoError(t, valid.Validate())

	cases := []struct {
		name   string
		mutate func(*Event)
		msg    string
	}{
		{"missing service", func(e *Event) { e.Service = "" }, "audit event requires service"},
		{"missing event type", func(e *Event) { e.EventType = "" }, "audit event requires event_type"},
		{"missing action", func(e *Event) { e.Action = "" }, "audit event requires action"},
		{"missing resource", func(e *Event) { e.Resource = "" }, "audit event requires resource"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := NewEvent("svc", EventTypePIIAccess, "read", "users/1")
			tc.mutate(&event)

			err := event.Validate()
			require.Error(t, err)
			assert.EqualError(t, err, tc.msg)
		})
	}

	t.Run("zero value fails validation", func(t *testing.T) {
		var event Event
		assert.Error(t, event.Validate())
	})
}

func TestConsoleAuditLogger(t *testing.T) {
	t.Run("nil logger is a no-op", func(t *testing.T) {
		logger := &ConsoleAuditLogger{}
		assert.NotPanics(t, func() {
			logger.Log(NewEvent("svc", EventTypePIIAccess, "read", "users/1"))
		})
	})

	t.Run("writes structured event line", func(t *testing.T) {
		buf := &bytes.Buffer{}
		encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
		core := zapcore.NewCore(encoder, zapcore.AddSync(buf), zap.InfoLevel)
		auditLogger := NewConsoleAuditLogger(zap.New(core))

		event := NewEvent("user-service", EventTypePIIDelete, "delete", "users/42").
			WithUser("42").
			WithActor("admin-2").
			WithFields("email").
			WithMetadata("reason", "gdpr")

		auditLogger.Log(event)

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		require.Len(t, lines, 1)

		var entry map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &entry))
		assert.Equal(t, "audit", entry["msg"])
		assert.Equal(t, "info", entry["level"])

		payload, ok := entry["event"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "user-service", payload["service"])
		assert.Equal(t, "pii_delete", payload["event_type"])
		assert.Equal(t, "42", payload["user_id"])
		assert.Equal(t, "admin-2", payload["actor_id"])
		assert.Equal(t, []interface{}{"email"}, payload["fields"])
		assert.Equal(t, map[string]interface{}{"reason": "gdpr"}, payload["metadata"])
	})
}

func TestAuditLoggerInterface(t *testing.T) {
	var _ Logger = (*ConsoleAuditLogger)(nil)
	var _ Logger = (*recordingLogger)(nil)

	sink := &recordingLogger{}
	event := NewEvent("svc", EventTypePIIAccess, "read", "users/1").WithUser("1")
	sink.Log(event)
	sink.Log(NewEvent("svc", EventTypePIIAccess, "read", "users/2").WithUser("2"))

	require.Len(t, sink.events, 2)
	assert.Equal(t, "1", sink.events[0].UserID)
	assert.Equal(t, "2", sink.events[1].UserID)
}

func TestEventTypeConstants(t *testing.T) {
	expected := []EventType{
		EventTypePIIAccess,
		EventTypePIIUpdate,
		EventTypePIIDelete,
		EventTypePIIExport,
		EventTypeAuthSuccess,
		EventTypeAuthFailure,
		EventTypeAdminAction,
	}

	values := make(map[EventType]bool, len(expected))
	for _, ct := range expected {
		assert.NotEmpty(t, ct)
		assert.False(t, values[ct], "duplicate event type constant: %s", ct)
		values[ct] = true
	}

	assert.Equal(t, EventType("pii_access"), EventTypePIIAccess)
	assert.Equal(t, EventType("admin_action"), EventTypeAdminAction)
}
