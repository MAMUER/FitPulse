// Package audit provides structured audit logging for 152-ФЗ compliance.
package audit

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// EventType defines the category of an auditable action.
type EventType string

const (
	// EventTypePIIAccess records access to personal data.
	EventTypePIIAccess EventType = "pii_access"
	// EventTypePIIUpdate records modification of personal data.
	EventTypePIIUpdate EventType = "pii_update"
	// EventTypePIIDelete records deletion of personal data.
	EventTypePIIDelete EventType = "pii_delete"
	// EventTypePIIExport records export of personal data.
	EventTypePIIExport EventType = "pii_export"
	// EventTypeAuthSuccess records successful authentication.
	EventTypeAuthSuccess EventType = "auth_success"
	// EventTypeAuthFailure records failed authentication.
	EventTypeAuthFailure EventType = "auth_failure"
	// EventTypeAdminAction records administrative action.
	EventTypeAdminAction EventType = "admin_action"
)

// FieldMask describes which PII fields were accessed or modified.
type FieldMask []string

// MarshalJSON implements custom JSON marshaling for FieldMask to ensure
// deterministic output and avoid nil slice serialization issues.
func (f FieldMask) MarshalJSON() ([]byte, error) {
	if f == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(f))
}

// Event represents a single audit log entry.
type Event struct {
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	EventType EventType  `json:"event_type"`
	UserID    string    `json:"user_id,omitempty"`
	ActorID   string    `json:"actor_id,omitempty"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Fields    FieldMask `json:"fields,omitempty"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// Logger abstracts audit log persistence.
type Logger interface {
	Log(event Event)
}

// ConsoleAuditLogger writes audit events to a zap.Logger.
type ConsoleAuditLogger struct {
	logger *zap.Logger
}

// NewConsoleAuditLogger creates a new console-backed audit logger.
func NewConsoleAuditLogger(logger *zap.Logger) *ConsoleAuditLogger {
	return &ConsoleAuditLogger{logger: logger}
}

// Log writes the audit event as a structured JSON line.
func (a *ConsoleAuditLogger) Log(event Event) {
	if a.logger == nil {
		return
	}
	a.logger.Info("audit", zap.Any("event", event))
}

// NewEvent creates a minimal audit event with timestamp.
func NewEvent(service string, eventType EventType, action, resource string) Event {
	return Event{
		Timestamp: time.Now().UTC(),
		Service:   service,
		EventType: eventType,
		Action:    action,
		Resource:  resource,
		Success:   true,
		Metadata:  make(map[string]interface{}),
	}
}

// WithUser sets the subject user ID.
func (e Event) WithUser(userID string) Event {
	e.UserID = userID
	return e
}

// WithActor sets the actor (admin/system) ID.
func (e Event) WithActor(actorID string) Event {
	e.ActorID = actorID
	return e
}

// WithFields sets the accessed/modified field names.
func (e Event) WithFields(fields ...string) Event {
	e.Fields = append(e.Fields, fields...)
	return e
}

// WithCorrelationID sets the correlation ID.
func (e Event) WithCorrelationID(correlationID string) Event {
	e.CorrelationID = correlationID
	return e
}

// WithError marks the event as failed with an error message.
func (e Event) WithError(err error) Event {
	if err != nil {
		e.Error = err.Error()
		e.Success = false
	}
	return e
}

// WithMetadata adds arbitrary key-value metadata.
func (e Event) WithMetadata(key string, value interface{}) Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// Validate ensures the event has required fields.
func (e Event) Validate() error {
	if e.Service == "" {
		return fmt.Errorf("audit event requires service")
	}
	if e.EventType == "" {
		return fmt.Errorf("audit event requires event_type")
	}
	if e.Action == "" {
		return fmt.Errorf("audit event requires action")
	}
	if e.Resource == "" {
		return fmt.Errorf("audit event requires resource")
	}
	return nil
}
