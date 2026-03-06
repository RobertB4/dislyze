package errlib

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppErrorError(t *testing.T) {
	tests := []struct {
		name           string
		appErr         *AppError
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "all fields set",
			appErr: &AppError{
				OriginalError: errors.New("db connection failed"),
				Message:       "something went wrong",
				StatusCode:    500,
				File:          "handler.go",
				Line:          42,
			},
			wantContains: []string{
				"AppError: handler.go:42",
				"UserMessage: something went wrong",
				"StatusCode: 500",
				"OriginalError: db connection failed",
			},
		},
		{
			name: "no message no status no original error",
			appErr: &AppError{
				File: "handler.go",
				Line: 10,
			},
			wantContains: []string{
				"AppError: handler.go:10",
			},
			wantNotContain: []string{
				"UserMessage",
				"StatusCode",
				"OriginalError",
			},
		},
		{
			name: "only message set",
			appErr: &AppError{
				Message: "bad request",
				File:    "auth.go",
				Line:    5,
			},
			wantContains: []string{
				"AppError: auth.go:5",
				"UserMessage: bad request",
			},
			wantNotContain: []string{
				"StatusCode",
				"OriginalError",
			},
		},
		{
			name: "only status code set",
			appErr: &AppError{
				StatusCode: 404,
				File:       "users.go",
				Line:       99,
			},
			wantContains: []string{
				"AppError: users.go:99",
				"StatusCode: 404",
			},
			wantNotContain: []string{
				"UserMessage",
				"OriginalError",
			},
		},
		{
			name: "only original error set",
			appErr: &AppError{
				OriginalError: errors.New("timeout"),
				File:          "db.go",
				Line:          1,
			},
			wantContains: []string{
				"AppError: db.go:1",
				"OriginalError: timeout",
			},
			wantNotContain: []string{
				"UserMessage",
				"StatusCode",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.appErr.Error()

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want)
			}
			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, result, notWant)
			}
		})
	}
}

func TestAppErrorImplementsErrorInterface(t *testing.T) {
	var err error = &AppError{
		Message:    "test",
		StatusCode: 400,
		File:       "test.go",
		Line:       1,
	}

	assert.NotEmpty(t, err.Error())
}

func TestAppErrorUnwrap(t *testing.T) {
	t.Run("returns original error", func(t *testing.T) {
		original := errors.New("original error")
		appErr := &AppError{OriginalError: original}

		assert.Equal(t, original, appErr.Unwrap())
	})

	t.Run("returns nil when no original error", func(t *testing.T) {
		appErr := &AppError{}

		assert.Nil(t, appErr.Unwrap())
	})
}

func TestNew(t *testing.T) {
	t.Run("captures caller file and line", func(t *testing.T) {
		appErr := New(errors.New("test"), 500, "test message")

		assert.NotEmpty(t, appErr.File)
		assert.NotEqual(t, "???", appErr.File)
		assert.True(t, strings.HasSuffix(appErr.File, "errlib_test.go"),
			"expected file to end with errlib_test.go, got %s", appErr.File)
		assert.Greater(t, appErr.Line, 0)
	})

	t.Run("sets all fields", func(t *testing.T) {
		original := errors.New("db error")
		appErr := New(original, 503, "service unavailable")

		assert.Equal(t, original, appErr.OriginalError)
		assert.Equal(t, 503, appErr.StatusCode)
		assert.Equal(t, "service unavailable", appErr.Message)
	})

	t.Run("nil original error", func(t *testing.T) {
		appErr := New(nil, 400, "bad request")

		assert.Nil(t, appErr.OriginalError)
		assert.Equal(t, 400, appErr.StatusCode)
		assert.Equal(t, "bad request", appErr.Message)
	})

	t.Run("empty message", func(t *testing.T) {
		appErr := New(errors.New("err"), 500, "")

		assert.Empty(t, appErr.Message)
		assert.NotContains(t, appErr.Error(), "UserMessage")
	})

	t.Run("zero status code", func(t *testing.T) {
		appErr := New(errors.New("err"), 0, "msg")

		assert.Equal(t, 0, appErr.StatusCode)
		assert.NotContains(t, appErr.Error(), "StatusCode")
	})
}

func TestIs(t *testing.T) {
	t.Run("matches same error", func(t *testing.T) {
		sentinel := errors.New("sentinel")
		assert.True(t, Is(sentinel, sentinel))
	})

	t.Run("matches wrapped error", func(t *testing.T) {
		sentinel := errors.New("sentinel")
		wrapped := fmt.Errorf("context: %w", sentinel)
		assert.True(t, Is(wrapped, sentinel))
	})

	t.Run("does not match different error", func(t *testing.T) {
		err1 := errors.New("error one")
		err2 := errors.New("error two")
		assert.False(t, Is(err1, err2))
	})

	t.Run("traverses AppError unwrap chain", func(t *testing.T) {
		sentinel := errors.New("not found")
		appErr := New(sentinel, 404, "not found")
		// Wrap the AppError in another error
		wrapped := fmt.Errorf("handler failed: %w", appErr)

		assert.True(t, Is(wrapped, sentinel))
	})
}

func TestAs(t *testing.T) {
	t.Run("extracts AppError from direct value", func(t *testing.T) {
		appErr := New(errors.New("db error"), 500, "internal error")

		var target *AppError
		assert.True(t, As(appErr, &target))
		assert.Equal(t, 500, target.StatusCode)
		assert.Equal(t, "internal error", target.Message)
	})

	t.Run("extracts AppError from wrapped error", func(t *testing.T) {
		appErr := New(errors.New("db error"), 503, "service down")
		wrapped := fmt.Errorf("handler: %w", appErr)

		var target *AppError
		assert.True(t, As(wrapped, &target))
		assert.Equal(t, 503, target.StatusCode)
		assert.Equal(t, "service down", target.Message)
	})

	t.Run("returns false for non-AppError", func(t *testing.T) {
		plainErr := errors.New("plain error")

		var target *AppError
		assert.False(t, As(plainErr, &target))
	})
}
