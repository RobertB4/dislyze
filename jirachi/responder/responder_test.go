package responder

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"dislyze/jirachi/errlib"

	"github.com/stretchr/testify/assert"
)

func TestRespondWithError(t *testing.T) {
	tests := []struct {
		name               string
		err                error
		wantStatusCode     int
		wantBody           string
		wantContentTypeSet bool
	}{
		{
			name:               "APIError with status code and detail",
			err:                &errlib.APIError{Status: http.StatusBadRequest, Detail: "invalid input"},
			wantStatusCode:     http.StatusBadRequest,
			wantBody:           `{"error":"invalid input"}`,
			wantContentTypeSet: true,
		},
		{
			name:               "APIError with status code but empty detail",
			err:                &errlib.APIError{Status: http.StatusNotFound},
			wantStatusCode:     http.StatusNotFound,
			wantBody:           "",
			wantContentTypeSet: false,
		},
		{
			name:               "plain error defaults to 500 with no body",
			err:                errors.New("something unexpected"),
			wantStatusCode:     http.StatusInternalServerError,
			wantBody:           "",
			wantContentTypeSet: false,
		},
		{
			name:               "APIError with zero status code defaults to 500",
			err:                &errlib.APIError{Status: 0, Detail: "oops"},
			wantStatusCode:     http.StatusInternalServerError,
			wantBody:           `{"error":"oops"}`,
			wantContentTypeSet: true,
		},
		{
			name:               "APIError with status code above 599 defaults to 500",
			err:                &errlib.APIError{Status: 999, Detail: "oops"},
			wantStatusCode:     http.StatusInternalServerError,
			wantBody:           `{"error":"oops"}`,
			wantContentTypeSet: true,
		},
		{
			name:               "APIError with status code below 100 defaults to 500",
			err:                &errlib.APIError{Status: 50, Detail: "oops"},
			wantStatusCode:     http.StatusInternalServerError,
			wantBody:           `{"error":"oops"}`,
			wantContentTypeSet: true,
		},
		{
			name:               "APIError at lower boundary status code 100",
			err:                &errlib.APIError{Status: 100, Detail: "continue"},
			wantStatusCode:     100,
			wantBody:           `{"error":"continue"}`,
			wantContentTypeSet: true,
		},
		{
			name:               "APIError at upper boundary status code 599",
			err:                &errlib.APIError{Status: 599, Detail: "custom error"},
			wantStatusCode:     599,
			wantBody:           `{"error":"custom error"}`,
			wantContentTypeSet: true,
		},
		{
			name:               "APIError with detail but no status code",
			err:                &errlib.APIError{Status: 0, Detail: "something went wrong"},
			wantStatusCode:     http.StatusInternalServerError,
			wantBody:           `{"error":"something went wrong"}`,
			wantContentTypeSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			RespondWithError(w, tt.err)

			assert.Equal(t, tt.wantStatusCode, w.Code)

			if tt.wantContentTypeSet {
				assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			} else {
				assert.Empty(t, w.Header().Get("Content-Type"))
			}

			if tt.wantBody != "" {
				var wantJSON, gotJSON map[string]string
				assert.NoError(t, json.Unmarshal([]byte(tt.wantBody), &wantJSON))
				assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &gotJSON))
				assert.Equal(t, wantJSON, gotJSON)
			} else {
				assert.Empty(t, w.Body.String())
			}
		})
	}
}

func TestRespondWithJSON(t *testing.T) {
	t.Run("valid payload with 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := map[string]string{"name": "Alice"}

		RespondWithJSON(w, http.StatusOK, payload)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

		var got map[string]string
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, "Alice", got["name"])
	})

	t.Run("nil payload writes no body", func(t *testing.T) {
		w := httptest.NewRecorder()

		RespondWithJSON(w, http.StatusNoContent, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
		assert.Empty(t, w.Body.String())
	})

	t.Run("status 201 with payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := map[string]int{"id": 42}

		RespondWithJSON(w, http.StatusCreated, payload)

		assert.Equal(t, http.StatusCreated, w.Code)

		var got map[string]int
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
		assert.Equal(t, 42, got["id"])
	})

	t.Run("empty struct payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := struct{}{}

		RespondWithJSON(w, http.StatusOK, payload)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{}`, w.Body.String())
	})

	t.Run("nested struct payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		type inner struct {
			Value string `json:"value"`
		}
		type outer struct {
			Inner inner `json:"inner"`
		}
		payload := outer{Inner: inner{Value: "test"}}

		RespondWithJSON(w, http.StatusOK, payload)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"inner":{"value":"test"}}`, w.Body.String())
	})

	t.Run("slice payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		payload := []string{"a", "b", "c"}

		RespondWithJSON(w, http.StatusOK, payload)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `["a","b","c"]`, w.Body.String())
	})
}
