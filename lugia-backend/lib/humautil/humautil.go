package humautil

import (
	"errors"
	"net/http"

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
func NewConfig(title, version string) huma.Config {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		return &APIError{status: status, Detail: msg}
	}

	config := huma.DefaultConfig(title, version)
	config.DocsPath = ""
	config.OpenAPIPath = ""
	return config
}

// MapError converts an errlib.AppError into a huma-compatible error.
// It preserves the existing logging behavior from responder.RespondWithError.
func MapError(err error) error {
	errlib.LogError(err)

	var ae *errlib.AppError
	if errors.As(err, &ae) {
		return &APIError{status: ae.StatusCode, Detail: ae.Message}
	}
	return &APIError{status: http.StatusInternalServerError}
}
