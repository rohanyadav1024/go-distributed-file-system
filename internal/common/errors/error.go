package errors

// This file defines the custom error type and helper functions for creating and wrapping errors.

type Error struct {
	Code   	Code
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap method to peel off nested errors
func (e *Error) Unwrap() error {
	return e.Cause
}

// Add a retryable method to determine if the error is transient
func (e *Error) Retryable() bool {
	// Todo: This is a simple heuristic. In a real implementation, you might want to be more specific about which codes are retryable. Add a configuration or mapping if needed.
	return e.Code == CodeUnavailable || e.Code == CodeTimeout || e.Code == CodeConflict
}

// Is method to compare errors based on their code, allowing for sentinel error checks
func (e *Error) Is(target error) bool {
    t, ok := target.(*Error)
    if !ok {
        return false
    }
    return e.Code == t.Code
}