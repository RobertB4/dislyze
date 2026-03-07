package errlib

import (
	stdErrors "errors"
	"log"
)

// APIError represents an HTTP error response. Through structural typing, it
// satisfies huma.StatusError (GetStatus() int + Error() string) without
// importing huma.
//
// When Detail is empty, it serializes to {} — the frontend treats this
// the same as a non-JSON error body (shows a generic toast).
type APIError struct {
	Status int    `json:"-"`
	Detail string `json:"error,omitempty"`
}

func (e *APIError) Error() string  { return e.Detail }
func (e *APIError) GetStatus() int { return e.Status }

// NewError logs the internal error and returns an APIError with the given
// status code. The client receives an empty error body and shows a generic
// toast. Use this for most errors.
//
// For errors where the client needs a specific message it can't determine
// on its own (e.g., "このメールアドレスは既に使用されています。"), use
// NewErrorWithDetail instead.
func NewError(err error, status int) error {
	LogError(err)
	return &APIError{Status: status}
}

// NewErrorWithDetail logs the internal error and returns an APIError with
// a user-visible detail message. Use this only when the client needs
// information that requires server knowledge.
func NewErrorWithDetail(err error, status int, detail string) error {
	LogError(err)
	return &APIError{Status: status, Detail: detail}
}

func LogError(err error) {
	log.Printf("%+v\n", err)
}

func Is(err, target error) bool {
	return stdErrors.Is(err, target)
}

func As(err error, target any) bool {
	return stdErrors.As(err, target)
}
