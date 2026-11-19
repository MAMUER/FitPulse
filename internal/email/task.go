package email

// QueueName is the RabbitMQ queue where email tasks are published for async
// delivery by the email-worker.
const QueueName = "emails"

// Email task types.
const (
	TypeVerification = "verification"
)

// Task is an email job published to RabbitMQ for asynchronous delivery.
// The email-worker consumes it and performs the actual SMTP send.
type Task struct {
	Type    string `json:"type"`     // one of TypeVerification, ...
	To      string `json:"to"`       // recipient email
	Token   string `json:"token"`    // verification token / code
	BaseURL string `json:"base_url"` // public base URL for links
}
