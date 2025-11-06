package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const SpecVersionV1 = "1.0"

type Envelope struct {
	SpecVersion string            `json:"spec_version"`
	Domain      string            `json:"domain"`
	EventType   string            `json:"event_type"`
	Source      string            `json:"source"`
	Timestamp   time.Time         `json:"timestamp"`
	Correlation map[string]string `json:"correlation,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Payload     json.RawMessage   `json:"payload"`
}

func ParseEnvelope(raw []byte) (Envelope, error) {
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if err := envelope.Validate(); err != nil {
		return Envelope{}, fmt.Errorf("invalid envelope: %w", err)
	}
	return envelope, nil
}

func (e Envelope) Validate() error {
	if e.SpecVersion == "" {
		return errors.New("spec_version is required")
	}
	if e.Domain == "" {
		return errors.New("domain is required")
	}
	if e.EventType == "" {
		return errors.New("event_type is required")
	}
	if len(e.Payload) == 0 {
		return errors.New("payload is required")
	}
	return nil
}

func (e Envelope) PayloadInto(target interface{}) error {
	if len(e.Payload) == 0 {
		return errors.New("envelope payload is empty")
	}
	if target == nil {
		return errors.New("target must not be nil")
	}
	if err := json.Unmarshal(e.Payload, target); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	return nil
}
