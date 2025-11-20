package events

import "time"

type StatResult struct {
	UserID      int       `json:"user_id"`
	SessionID   string    `json:"session_id"`
	EventType   string    `json:"event_type"`
	Anomaly     bool      `json:"anomaly"`
	AnomalyType string    `json:"anomaly_type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
}
