package events

import (
	"errors"
	"fmt"
	"time"
)

const DomainUserActivity = "user_activity"

type ActivityType string

const (
	ActivityLogin         ActivityType = "login"
	ActivityPayment       ActivityType = "payment"
	ActivityLogout        ActivityType = "logout"
	ActivityFailedLogin   ActivityType = "failed_login"
	ActivityPasswordReset ActivityType = "password_reset"
	ActivityOther         ActivityType = "other"
)

type UserMetadata struct {
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
	Country   string `json:"country"`
}

type UserActivityPayload struct {
	UserID     int                    `json:"user_id"`
	Type       ActivityType           `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	SessionID  string                 `json:"session_id"`
	Metadata   UserMetadata           `json:"metadata"`
	Additional map[string]interface{} `json:"additional,omitempty"`
}

func (p *UserActivityPayload) Validate() error {
	if p == nil {
		return errors.New("payload must not be nil")
	}
	if p.UserID == 0 {
		return errors.New("user_id must be set")
	}
	if p.Type == "" {
		return errors.New("type must be set")
	}
	if p.Timestamp.IsZero() {
		return errors.New("timestamp must be set")
	}
	if p.Additional == nil {
		p.Additional = map[string]interface{}{}
	}
	return nil
}

func (e Envelope) UserActivityPayload() (UserActivityPayload, error) {
	if e.Domain != DomainUserActivity {
		return UserActivityPayload{}, fmt.Errorf("expected domain %q, got %q", DomainUserActivity, e.Domain)
	}

	var payload UserActivityPayload
	if err := e.PayloadInto(&payload); err != nil {
		return UserActivityPayload{}, err
	}
	if err := payload.Validate(); err != nil {
		return UserActivityPayload{}, fmt.Errorf("invalid user activity payload: %w", err)
	}
	return payload, nil
}
