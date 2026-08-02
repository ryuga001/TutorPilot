package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypeEmailRequested = "notification.email.requested"

	TypeOTPEmailRequested = "notification.email.otp"
)

const CurrentVersion = 1

type Event struct {
	ID         uuid.UUID       `json:"id"`
	Type       string          `json:"type"`
	Version    int             `json:"version"`
	CustomerID int             `json:"customer_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

type EmailRequested struct {
	To           string            `json:"to"`
	TemplateName string            `json:"template_name"`
	Vars         map[string]string `json:"vars"`
}

func NewEmail(eventType string, customerID int, now time.Time, p EmailRequested) (Event, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return Event{}, err
	}
	return Event{
		ID:         uuid.New(),
		Type:       eventType,
		Version:    CurrentVersion,
		CustomerID: customerID,
		OccurredAt: now,
		Payload:    payload,
	}, nil
}

func Redactable(eventType string) bool {
	return eventType == TypeOTPEmailRequested
}

const redactedPlaceholder = "[redacted]"

func Redact(evt Event) Event {
	out := evt

	var p EmailRequested
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		out.Payload = json.RawMessage(`{"redacted":true,"reason":"payload unparseable"}`)
		return out
	}

	for k := range p.Vars {
		p.Vars[k] = redactedPlaceholder
	}

	redacted, err := json.Marshal(p)
	if err != nil {
		out.Payload = json.RawMessage(`{"redacted":true,"reason":"payload unmarshalable"}`)
		return out
	}
	out.Payload = redacted
	return out
}

func ForDLQ(evt Event) Event {
	if Redactable(evt.Type) {
		return Redact(evt)
	}
	return evt
}
