package adapters

import (
	"encoding/json"
	"fmt"
	"time"

	"synthetic-data-generator/internal/env"
	contracts "user-event-analisys/contracts/events"
)

func UserActivityToEnvelope(payload contracts.UserActivityPayload, sessionID string, metadata map[string]string) (contracts.Envelope, error) {
	if sessionID != "" && payload.SessionID == "" {
		payload.SessionID = sessionID
	}
	if payload.Additional == nil {
		payload.Additional = map[string]interface{}{}
	}
	if err := payload.Validate(); err != nil {
		return contracts.Envelope{}, fmt.Errorf("validate payload: %w", err)
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return contracts.Envelope{}, fmt.Errorf("marshal payload: %w", err)
	}

	envelope := contracts.Envelope{
		SpecVersion: contracts.SpecVersionV1,
		Domain:      contracts.DomainUserActivity,
		EventType:   string(payload.Type),
		Source:      env.GetEnvString("SOURCE", "synthetic-data-generator"),
		Timestamp:   time.Now().UTC(),
		Payload:     rawPayload,
	}

	if len(payload.SessionID) > 0 {
		envelope.Correlation = map[string]string{
			"session_id": payload.SessionID,
		}
	}

	if len(metadata) > 0 {
		envelope.Metadata = make(map[string]string, len(metadata))
		for k, v := range metadata {
			envelope.Metadata[k] = v
		}
	}

	return envelope, nil
}
