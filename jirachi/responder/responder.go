package responder

import (
	"encoding/json"
	"net/http"

	"dislyze/jirachi/errlib"
	stdErrors "errors"
)

func RespondWithError(w http.ResponseWriter, err error) {
	var ae *errlib.APIError
	if !stdErrors.As(err, &ae) {
		// Plain error — log it and create a generic 500 APIError
		errlib.LogError(err)
		ae = &errlib.APIError{Status: http.StatusInternalServerError}
	}
	// APIError created by NewError/NewErrorWithDetail was already logged at creation time

	responseStatusCode := http.StatusInternalServerError
	if ae.Status >= 100 && ae.Status <= 599 {
		responseStatusCode = ae.Status
	}

	if ae.Detail != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	w.WriteHeader(responseStatusCode)

	if ae.Detail != "" {
		if err := json.NewEncoder(w).Encode(map[string]string{"error": ae.Detail}); err != nil {
			errlib.LogError(err)
		}
	}
}

func RespondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			errlib.LogError(err)
			if statusCode < 400 {
				http.Error(w, "", http.StatusInternalServerError)
			}
		}
	}
}
