package event

import (
	"encoding/json"
	"fmt"
	"time"
)

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
	Metadata   Metadata               `json:"metadata"`
	Additional map[string]interface{} `json:"additional"`
}

type Metadata struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Country   string `json:"country"`
}

func (m *Metadata) UnmarshallJSON(data []byte) error {
	var v [3]string
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}

	m.IP = v[0]
	m.UserAgent = v[1]
	m.Country = v[2]
	return nil
}

func ParseEvent(e []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(e, &event); err != nil {
		return Event{}, err
	}

	fmt.Printf("%#v\n", event)
	return event, nil
}
