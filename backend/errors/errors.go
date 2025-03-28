package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// AppError represents an application error with stack trace
type AppError struct {
	Err     error
	Stack   string
	Message string
	Code    int
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Err.Error()
}

// New creates a new AppError with stack trace
func New(err error, message string, code int) *AppError {
	stack := make([]byte, 4096)
	stack = stack[:runtime.Stack(stack, false)]
	// Skip the first few lines that are about the stack trace itself
	stackLines := strings.Split(string(stack), "\n")
	if len(stackLines) > 3 {
		stackLines = stackLines[3:]
	}

	return &AppError{
		Err:     err,
		Stack:   strings.Join(stackLines, "\n"),
		Message: message,
		Code:    code,
	}
}

// LogError logs the error with stack trace
func LogError(err error) {
	if appErr, ok := err.(*AppError); ok {
		fmt.Printf("Error: %s\nStack trace:\n%s\n", appErr.Error(), appErr.Stack)
	} else {
		fmt.Printf("Error: %v\n", err)
	}
}
