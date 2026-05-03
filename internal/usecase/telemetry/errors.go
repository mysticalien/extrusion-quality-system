package telemetry

import (
	"errors"
)

// ValidationError is returned when incoming telemetry data is invalid.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// IsValidationError checks whether an error is caused by invalid telemetry input.
func IsValidationError(err error) bool {
	var validationError ValidationError

	return errors.As(err, &validationError)
}
