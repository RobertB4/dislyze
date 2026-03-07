package errlib

import (
	"bytes"
	"errors"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIErrorError(t *testing.T) {
	t.Run("returns detail as error string", func(t *testing.T) {
		ae := &APIError{Status: 400, Detail: "bad request"}
		assert.Equal(t, "bad request", ae.Error())
	})

	t.Run("returns empty string when no detail", func(t *testing.T) {
		ae := &APIError{Status: 500}
		assert.Equal(t, "", ae.Error())
	})
}

func TestAPIErrorGetStatus(t *testing.T) {
	t.Run("returns status code", func(t *testing.T) {
		ae := &APIError{Status: 404}
		assert.Equal(t, 404, ae.GetStatus())
	})
}

func TestNewError(t *testing.T) {
	t.Run("logs error and returns APIError with status", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		log.SetFlags(0)
		t.Cleanup(func() {
			log.SetOutput(nil)
			log.SetFlags(log.LstdFlags)
		})

		err := NewError(errors.New("db timeout"), 500)

		var ae *APIError
		assert.True(t, errors.As(err, &ae))
		assert.Equal(t, 500, ae.GetStatus())
		assert.Empty(t, ae.Detail)
		assert.Contains(t, buf.String(), "db timeout")
	})
}

func TestNewErrorWithDetail(t *testing.T) {
	t.Run("logs error and returns APIError with status and detail", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		log.SetFlags(0)
		t.Cleanup(func() {
			log.SetOutput(nil)
			log.SetFlags(log.LstdFlags)
		})

		err := NewErrorWithDetail(errors.New("duplicate email"), 409, "このメールアドレスは既に使用されています。")

		var ae *APIError
		assert.True(t, errors.As(err, &ae))
		assert.Equal(t, 409, ae.GetStatus())
		assert.Equal(t, "このメールアドレスは既に使用されています。", ae.Detail)
		assert.Contains(t, buf.String(), "duplicate email")
	})
}

func TestLogError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(nil)
		log.SetFlags(log.LstdFlags)
	})

	LogError(errors.New("something broke"))

	assert.Contains(t, buf.String(), "something broke")
}
