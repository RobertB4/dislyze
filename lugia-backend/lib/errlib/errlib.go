package errlib

import (
	stdErrors "errors"
	"fmt"
	"log"
	"runtime"
	"strings"
)

type AppError struct {
	OriginalError error
	Message       string
	StatusCode    int
	File          string
	Line          int
}

func (e *AppError) Error() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("AppError: %s:%d", e.File, e.Line))
	if e.Message != "" {
		sb.WriteString(fmt.Sprintf(" | UserMessage: %s", e.Message))
	}
	if e.StatusCode != 0 {
		sb.WriteString(fmt.Sprintf(" | StatusCode: %d", e.StatusCode))
	}
	if e.OriginalError != nil {
		sb.WriteString(fmt.Sprintf(" | OriginalError: %v", e.OriginalError))
	}
	return sb.String()
}

func (e *AppError) Unwrap() error {
	return e.OriginalError
}

func New(originalError error, statusCode int, userMessage string) *AppError {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	}
	return &AppError{
		OriginalError: originalError,
		StatusCode:    statusCode,
		Message:       userMessage,
		File:          file,
		Line:          line,
	}
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
