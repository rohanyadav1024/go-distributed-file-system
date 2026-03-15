package errors

// Error represents an application error with code, message, and optional cause.
type Error struct {
	Code    Code
	Message string
	Cause   error
}

// Error returns the error message, including the wrapped cause when present.
func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the wrapped cause.
func (e *Error) Unwrap() error {
	return e.Cause
}

// Retryable reports whether the error is usually safe to retry.
func (e *Error) Retryable() bool {
	return e.Code == CodeUnavailable || e.Code == CodeTimeout || e.Code == CodeConflict
}

// Is reports whether target has the same application error code.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}
