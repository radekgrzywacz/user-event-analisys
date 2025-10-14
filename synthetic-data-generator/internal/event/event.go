package event

import "time"

type EventType string

const (
	EventLogin         EventType = "login"
	EventPayment       EventType = "payment"
	EventLogout        EventType = "logout"
	EventFailedLogin   EventType = "failedLogin"
	EventPasswordReset EventType = "passwordReset"
	EventOther         EventType = "other"
)

type Event struct {
	UserId     int                    `json:"user_id"`
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	SessionId  string                 `json:"session_id"`
	Metadata   Metadata               `json:"metadata"`
	Additional map[string]interface{} `json:"additional"`
}

type Metadata struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Country   string `json:"country"`
}
