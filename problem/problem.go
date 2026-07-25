package problem

import "fmt"

// Error represents a structured error response for the application.
//
// It is based on RFC 7807 (Problem Details for HTTP APIs). (https://datatracker.ietf.org/doc/html/rfc7807)
type Error struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Errors   any    `json:"errors,omitempty"`

	cause error
}

// ErrorFunc is a function type that generates an Error with optional details.
type ErrorFunc func(details ...string) *Error

// Register creates a new ErrorFunc with the provided title, type, status, and optional detail.
//
// Example usage:
//
//	var ErrNotFound = problem.Register("Not Found", "not_found", 404)
//	err := ErrNotFound("The requested resource was not found.")
func Register(title, typeName string, status int, detail ...string) ErrorFunc {
	return func(details ...string) *Error {
		useDetail := ""
		if len(details) > 0 {
			useDetail = details[0]
		} else if len(detail) > 0 {
			useDetail = detail[0]
		}
		return New(title, typeName, status, useDetail)
	}
}

// New creates a new Error with the provided title, type, status, and optional detail.
func New(title, typeName string, status int, detail ...string) *Error {
	var errDetail string
	if len(detail) > 0 {
		errDetail = detail[0]
	}

	return &Error{
		Type:   typeName,
		Title:  title,
		Status: status,
		Detail: errDetail,
	}
}

// Error implements the error interface for Error, returning a string representation of the error.
func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Title, e.Detail)
	}
	return e.Title
}

// Unwrap lets `errors.Is` and `errors.As` work.
func (e *Error) Unwrap() error {
	return e.cause
}

// Is reports whether target is a *Error with the same Type, so `errors.Is` can match
// registered errors (e.g. errors.Is(err, ErrNotFound())) even though Register and New
// return a distinct *Error instance on every call.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// WithDetail sets the Detail field of the Error and returns the modified error for chaining.
func (e *Error) WithDetail(detail string) *Error {
	e.Detail = detail
	return e
}

// WithErrors sets the Errors field of the Error and returns the modified error for chaining.
func (e *Error) WithErrors(errors any) *Error {
	e.Errors = errors
	return e
}

// WithCause sets the cause of the Error and returns the modified error for chaining.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// WithInstance sets the Instance field of the Error and returns the modified error for chaining.
func (e *Error) WithInstance(instance string) *Error {
	e.Instance = instance
	return e
}
