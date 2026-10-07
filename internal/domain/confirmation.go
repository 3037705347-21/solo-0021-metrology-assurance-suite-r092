package domain

import (
	"fmt"
	"strings"
	"time"
)

// RiskLevel is fixed when a case is opened and never changes afterwards.
// Snapshotting it onto the case and every confirmation means later policy or
// personnel changes cannot rewrite what an existing confirmation relied on.
type RiskLevel string

const (
	RiskStandard RiskLevel = "standard"
	RiskHigh     RiskLevel = "high"
)

// ConfirmationPolicyVersion identifies the independent-confirmation rule set.
// It is snapshotted onto each confirmation so the rule basis stays traceable.
const ConfirmationPolicyVersion = "independent-confirmation-v1"

func (r RiskLevel) Valid() bool {
	switch r {
	case RiskStandard, RiskHigh:
		return true
	default:
		return false
	}
}

// RequiredIndependentConfirmations is the number of distinct parties other
// than the recording technician that must confirm the sealed measurement.
func (r RiskLevel) RequiredIndependentConfirmations() int {
	if r == RiskHigh {
		return 1
	}
	return 0
}

// NormalizePerson canonicalizes a person identity for same-person comparisons
// so whitespace or casing cannot bypass the independence rule.
func NormalizePerson(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// IndependentConfirmation is an append-only, immutable record that a second
// person independently checked a specific accepted measurement. It snapshots
// the risk requirement, the policy version, and the recording technician, so
// later personnel or requirement changes cannot alter an existing record.
type IndependentConfirmation struct {
	ID                    string    `json:"id"`
	CaseID                string    `json:"case_id"`
	MeasurementID         string    `json:"measurement_id"`
	Confirmer             string    `json:"confirmer"`
	Technician            string    `json:"technician"`
	Basis                 string    `json:"basis"`
	RiskLevel             RiskLevel `json:"risk_level"`
	RequiredConfirmations int       `json:"required_confirmations"`
	PolicyVersion         string    `json:"policy_version"`
	CreatedAt             time.Time `json:"created_at"`
}

func NewIndependentConfirmation(id, caseID, measurementID, confirmer, technician, basis string, risk RiskLevel, at time.Time) (IndependentConfirmation, error) {
	item := IndependentConfirmation{
		ID:                    strings.TrimSpace(id),
		CaseID:                strings.TrimSpace(caseID),
		MeasurementID:         strings.TrimSpace(measurementID),
		Confirmer:             strings.TrimSpace(confirmer),
		Technician:            strings.TrimSpace(technician),
		Basis:                 strings.TrimSpace(basis),
		RiskLevel:             risk,
		RequiredConfirmations: risk.RequiredIndependentConfirmations(),
		PolicyVersion:         ConfirmationPolicyVersion,
		CreatedAt:             at.UTC(),
	}
	if err := item.Validate(); err != nil {
		return IndependentConfirmation{}, err
	}
	return item, nil
}

func (c IndependentConfirmation) Validate() error {
	if c.ID == "" || c.CaseID == "" || c.MeasurementID == "" || c.Confirmer == "" || c.Technician == "" || c.Basis == "" {
		return fmt.Errorf("%w: confirmation id, case, measurement, confirmer, technician, and basis are required", ErrInvalidInput)
	}
	if !c.RiskLevel.Valid() {
		return fmt.Errorf("%w: unsupported risk level %q", ErrInvalidInput, c.RiskLevel)
	}
	if c.RequiredConfirmations != c.RiskLevel.RequiredIndependentConfirmations() {
		return fmt.Errorf("%w: confirmation requirement snapshot does not match risk level", ErrInvalidInput)
	}
	if c.PolicyVersion != ConfirmationPolicyVersion {
		return fmt.Errorf("%w: confirmation policy version mismatch", ErrInvalidInput)
	}
	if c.CreatedAt.IsZero() {
		return fmt.Errorf("%w: confirmation time is required", ErrInvalidInput)
	}
	if NormalizePerson(c.Confirmer) == NormalizePerson(c.Technician) {
		return fmt.Errorf("%w: confirmer %q must be a different person from the recording technician %q", ErrIndependentPartyRequired, c.Confirmer, c.Technician)
	}
	return nil
}
