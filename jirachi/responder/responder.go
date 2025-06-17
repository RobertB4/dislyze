package responder

import (
	"encoding/json"
	"net/http"

	"dislyze/jirachi/errlib"
	stdErrors "errors"
)

func RespondWithError(w http.ResponseWriter, err error) {
	var ae *errlib.AppError
	if !stdErrors.As(err, &ae) {
		loggedAppError := errlib.New(err, http.StatusInternalServerError, err.Error())
		errlib.LogError(loggedAppError)
	} else {
		errlib.LogError(err)
	}

	responseStatusCode := http.StatusInternalServerError
	var responseUserMessage string

	if stdErrors.As(err, &ae) {
		if ae.StatusCode >= 100 && ae.StatusCode <= 599 {
			responseStatusCode = ae.StatusCode
		}
		if ae.Message != "" {
			responseUserMessage = ae.Message
		}
	}

	if responseUserMessage != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}

	w.WriteHeader(responseStatusCode)

	if responseUserMessage != "" {
		if err := json.NewEncoder(w).Encode(map[string]string{"error": responseUserMessage}); err != nil {
			errlib.LogError(errlib.New(err, http.StatusInternalServerError, "failed to encode error response"))
		}
	}
}

func RespondWithJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {

			encodingErr := errlib.New(err, http.StatusInternalServerError, "")
			errlib.LogError(encodingErr)
			// At this point, headers (including status) have likely been sent.
			// If the original status was a success one, we can't reliably change it.
			// This http.Error is a best effort to inform the client if the response stream is still writable.
			if statusCode < 400 {
				http.Error(w, "", http.StatusInternalServerError)
			}
		}
	}
}
