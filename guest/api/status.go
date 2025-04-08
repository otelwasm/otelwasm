package api

// Status represents the status from a plugin function.
type Status struct {
	Code   StatusCode
	Reason string
}

type StatusCode int32

// These are predefined codes used in a Status.
const (
	// Completed without errors.
	StatusCodeSuccess StatusCode = iota
	// Exited with unexpected errors.
	StatusCodeError
)
