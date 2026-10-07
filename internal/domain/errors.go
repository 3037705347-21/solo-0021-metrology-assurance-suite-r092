package domain

import "errors"

var (
	ErrInvalidInput               = errors.New("invalid input")
	ErrNotFound                   = errors.New("resource not found")
	ErrDuplicateAssetTag          = errors.New("duplicate asset tag")
	ErrActiveCaseExists           = errors.New("device already has an unfinished case")
	ErrStateConflict              = errors.New("resource state conflict")
	ErrSealAlreadyCreated         = errors.New("case already has an evidence seal")
	ErrConfirmationRequired       = errors.New("independent confirmation is required before sealing")
	ErrConfirmationExists         = errors.New("measurement already has a confirmation")
	ErrConfirmationNotIndependent = errors.New("confirmer must differ from the recording technician")
)

func IsKnownError(err error) bool {
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrDuplicateAssetTag) ||
		errors.Is(err, ErrActiveCaseExists) ||
		errors.Is(err, ErrStateConflict) ||
		errors.Is(err, ErrSealAlreadyCreated) ||
		errors.Is(err, ErrConfirmationRequired) ||
		errors.Is(err, ErrConfirmationExists) ||
		errors.Is(err, ErrConfirmationNotIndependent)
}
