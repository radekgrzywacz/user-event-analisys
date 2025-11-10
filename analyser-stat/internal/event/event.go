package event

import (
	"fmt"
	"time"

	contracts "user-event-analisys/contracts/events"
)

type EventType = contracts.ActivityType

const (
	EventLogin         EventType = contracts.ActivityLogin
	EventPayment       EventType = contracts.ActivityPayment
	EventLogout        EventType = contracts.ActivityLogout
	EventFailedLogin   EventType = contracts.ActivityFailedLogin
	EventPasswordReset EventType = contracts.ActivityPasswordReset
	EventOther         EventType = contracts.ActivityOther
)

type Metadata = contracts.UserMetadata

type Event struct {
	Envelope   contracts.Envelope     `json:"envelope"`
	UserId     int                    `json:"user_id"`
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   Metadata               `json:"metadata"`
	SessionId  string                 `json:"session_id"`
	Additional map[string]interface{} `json:"additional"`
}

func ParseEvent(envelope contracts.Envelope) (Event, error) {
	if envelope.Domain != contracts.DomainUserActivity {
		return Event{}, fmt.Errorf("unsupported domain: %s", envelope.Domain)
	}

	payload, err := envelope.UserActivityPayload()
	if err != nil {
		return Event{}, fmt.Errorf("decode user activity payload: %w", err)
	}

	if payload.Additional == nil {
		payload.Additional = map[string]interface{}{}
	}

	return Event{
		Envelope:   envelope,
		UserId:     payload.UserID,
		Type:       payload.Type,
		Timestamp:  payload.Timestamp,
		Metadata:   payload.Metadata,
		SessionId:  payload.SessionID,
		Additional: payload.Additional,
	}, nil
}
