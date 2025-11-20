package events

import "time"

type MLResult struct {
	UserID       int       `json:"user_id"`
	SessionID    string    `json:"session_id"`
	Timestamp    time.Time `json:"timestamp"`
	Anomaly      bool      `json:"anomaly"`
	Score        float64   `json:"score"`
	Threshold    float64   `json:"threshold"`
	EventCount   int       `json:"event_count"`
	UniqueEvents int       `json:"unique_events"`
	Source       string    `json:"source"`
}
