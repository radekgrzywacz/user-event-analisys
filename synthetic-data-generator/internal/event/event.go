package event

import (
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
	UserId     int                    `json:"user_id"`
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	SessionId  string                 `json:"session_id"`
	Metadata   Metadata               `json:"metadata"`
	Additional map[string]interface{} `json:"additional"`
}
