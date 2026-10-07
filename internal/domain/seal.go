package domain

import (
	"fmt"
	"strings"
	"time"
)

type EvidenceSeal struct {
	ID              string    `json:"id"`
	CaseID          string    `json:"case_id"`
	DeviceID        string    `json:"device_id"`
	MeasurementID   string    `json:"measurement_id"`
	Reviewer        string    `json:"reviewer"`
	Summary         string    `json:"summary"`
	ConfirmationIDs []string  `json:"confirmation_ids"`
	SealedAt        time.Time `json:"sealed_at"`
}

func NewEvidenceSeal(id, caseID, deviceID, measurementID, reviewer, summary string, confirmationIDs []string, at time.Time) (EvidenceSeal, error) {
	item := EvidenceSeal{
		ID:              strings.TrimSpace(id),
		CaseID:          strings.TrimSpace(caseID),
		DeviceID:        strings.TrimSpace(deviceID),
		MeasurementID:   strings.TrimSpace(measurementID),
		Reviewer:        strings.TrimSpace(reviewer),
		Summary:         strings.TrimSpace(summary),
		ConfirmationIDs: dedupeAndTrim(confirmationIDs),
		SealedAt:        at.UTC(),
	}
	if err := item.Validate(); err != nil {
		return EvidenceSeal{}, err
	}
	return item, nil
}

func (s EvidenceSeal) Validate() error {
	if s.ID == "" || s.CaseID == "" || s.DeviceID == "" || s.MeasurementID == "" || s.Reviewer == "" || s.Summary == "" {
		return fmt.Errorf("%w: seal id, case, device, measurement, reviewer, and summary are required", ErrInvalidInput)
	}
	if s.SealedAt.IsZero() {
		return fmt.Errorf("%w: seal time is required", ErrInvalidInput)
	}
	return nil
}

func dedupeAndTrim(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
