package errors

// This file defines the custom error type and helper functions for creating and wrapping errors.

// New creates a new Error with the given code and message.
func New(code Code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// sentinel errors for common cases
var (
    ErrNotFound        = &Error{Code: CodeNotFound}
    ErrInvalidArgument = &Error{Code: CodeInvalidArgument}
    ErrAlreadyExists   = &Error{Code: CodeAlreadyExists}
)

// Wrap creates a new Error that wraps an existing error with additional context.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// From converts a standard error into our custom Error type.
// If the error is already of type *Error, it returns it directly.
// Otherwise, it wraps it with CodeInternal.
func From(err error) *Error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return e
	}

	return &Error{
		Code:    CodeInternal,
		Message: err.Error(),
		Cause:   err,
	}
}