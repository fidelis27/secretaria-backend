package event

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

type Event struct {
	EventID       string          `json:"eventId"`
	Type          string          `json:"type"`
	Version       int             `json:"version"`
	Source        string          `json:"source"`
	CorrelationID string          `json:"correlationId"`
	OccurredAt    time.Time       `json:"occurredAt"`
	Payload       json.RawMessage `json:"payload"`
}

func New(eventType string, source string, correlationID string, payload any) (Event, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		correlationID = newID()
	}
	return Event{
		EventID:       newID(),
		Type:          strings.TrimSpace(eventType),
		Version:       1,
		Source:        strings.TrimSpace(source),
		CorrelationID: correlationID,
		OccurredAt:    time.Now().UTC(),
		Payload:       encodedPayload,
	}, nil
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
