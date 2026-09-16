package event

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

type Event struct {
	EventID        string          `json:"eventId"`
	Type           string          `json:"type"`
	Version        int             `json:"version"`
	Source         string          `json:"source"`
	CorrelationID  string          `json:"correlationId"`
	InstitutionIDs []string        `json:"institutionIds,omitempty"`
	OccurredAt     time.Time       `json:"occurredAt"`
	Payload        json.RawMessage `json:"payload"`
}

func New(eventType string, source string, correlationID string, payload any) (Event, error) {
	return NewForInstitutions(eventType, source, correlationID, nil, payload)
}

func NewForInstitution(eventType string, source string, correlationID string, institutionID string, payload any) (Event, error) {
	return NewForInstitutions(eventType, source, correlationID, []string{institutionID}, payload)
}

func NewForInstitutions(eventType string, source string, correlationID string, institutionIDs []string, payload any) (Event, error) {
	encodedPayload, err := json.Marshal(payload)
	if err != nil {
		return Event{}, err
	}
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		correlationID = newID()
	}
	return Event{
		EventID:        newID(),
		Type:           strings.TrimSpace(eventType),
		Version:        1,
		Source:         strings.TrimSpace(source),
		CorrelationID:  correlationID,
		InstitutionIDs: cleanInstitutionIDs(institutionIDs),
		OccurredAt:     time.Now().UTC(),
		Payload:        encodedPayload,
	}, nil
}

func cleanInstitutionIDs(institutionIDs []string) []string {
	seen := make(map[string]struct{}, len(institutionIDs))
	result := make([]string, 0, len(institutionIDs))
	for _, institutionID := range institutionIDs {
		institutionID = strings.TrimSpace(institutionID)
		if institutionID == "" {
			continue
		}
		if _, exists := seen[institutionID]; exists {
			continue
		}
		seen[institutionID] = struct{}{}
		result = append(result, institutionID)
	}
	return result
}

func newID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}
