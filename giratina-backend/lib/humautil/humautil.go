package humautil

import (
	"log"

	"dislyze/jirachi/errlib"

	"github.com/danielgtaylor/huma/v2"
)

// APIError matches the existing error response format: {"error": "message"}.
// When Detail is empty, it serializes to {} — the frontend treats this the
// same as a non-JSON error body (shows a generic toast).
type APIError struct {
	status int
	Detail string `json:"error,omitempty"`
}

func (e *APIError) Error() string  { return e.Detail }
func (e *APIError) GetStatus() int { return e.status }

// NewConfig creates a shared huma config. All huma API instances should use
// the same config so they contribute operations to the same OpenAPI spec.
// Docs and spec serving are disabled — the spec is generated offline via
// cmd/openapi and committed as openapi.json.
//
// Must be called exactly once per process (sets the global huma.NewError).
func NewConfig(title, version string) huma.Config {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		if len(errs) > 0 {
			log.Printf("huma error %d: %s (details: %v)", status, msg, errs)
		}
		return &APIError{status: status, Detail: msg}
	}

	config := huma.DefaultConfig(title, version)
	config.DocsPath = ""
	config.OpenAPIPath = ""
	return config
}

// NewError logs the internal error and returns a huma-compatible error with
// no user-visible detail. The client receives an empty error body and shows
// a generic toast. Use this for most errors.
//
// For errors where the client needs a specific message it can't determine
// on its own (e.g., "このメールアドレスは既に使用されています。"), use
// NewErrorWithDetail instead.
func NewError(err error, status int) error {
	errlib.LogError(err)
	return &APIError{status: status}
}

// NewErrorWithDetail logs the internal error and returns a huma-compatible
// error with a user-visible detail message. Use this only when the client
// needs information that requires server knowledge (e.g., "an account with
// this email already exists").
//
// For most errors where a generic toast is sufficient, use NewError instead.
func NewErrorWithDetail(err error, status int, detail string) error {
	errlib.LogError(err)
	return &APIError{status: status, Detail: detail}
}
