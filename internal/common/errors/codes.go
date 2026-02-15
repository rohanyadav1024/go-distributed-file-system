package errors

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