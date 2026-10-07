package domain

import (
	"fmt"
	"strings"
	"time"
)

// Confirmation is the immutable record that a second person independently
// confirmed one specific measurement. Confirmations are append-only: the
// service exposes no update or delete path, so later personnel or
// requirement changes cannot rewrite a confirmation that already exists.
type Confirmation struct {
	ID            string    `json:"id"`
	CaseID        string    `json:"case_id"`
	MeasurementID string    `json:"measurement_id"`
	Confirmer     string    `json:"confirmer"`
	Basis         string    `json:"basis"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewConfirmation(id, caseID, measurementID, confirmer, basis string, at time.Time) (Confirmation, error) {
	item := Confirmation{
		ID:            strings.TrimSpace(id),
		CaseID:        strings.TrimSpace(caseID),
		MeasurementID: strings.TrimSpace(measurementID),
		Confirmer:     strings.TrimSpace(confirmer),
		Basis:         strings.TrimSpace(basis),
		CreatedAt:     at.UTC(),
	}
	if err := item.Validate(); err != nil {
		return Confirmation{}, err
	}
	return item, nil
}

func (c Confirmation) Validate() error {
	if c.ID == "" || c.CaseID == "" || c.MeasurementID == "" || c.Confirmer == "" || c.Basis == "" {
		return fmt.Errorf("%w: confirmation id, case, measurement, confirmer, and basis are required", ErrInvalidInput)
	}
	if c.CreatedAt.IsZero() {
		return fmt.Errorf("%w: confirmation time is required", ErrInvalidInput)
	}
	return nil
}
