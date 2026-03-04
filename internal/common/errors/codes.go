package errors

// This file defines the error codes used throughout the application. These codes can be used to categorize errors and make it easier to handle them appropriately in different contexts.

type Code string

const (
	CodeInternal   Code = "internal"
	CodeInvalidArgument Code = "invalid_argument"
	CodeNotFound   Code = "not_found"
	CodeAlreadyExists Code = "already_exists"
	CodeUnauthenticated Code = "unauthenticated"
	CodePermissionDenied Code = "permission_denied"
	CodeUnavailable Code = "unavailable"
	CodeIntegrityViolation Code = "integrity_violation"
	CodeConflict Code = "conflict"
	CodeTimeout Code = "timeout"
)