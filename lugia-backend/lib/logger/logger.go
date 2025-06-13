package logger

import (
	"log"
	"time"
)

type AuthEvent struct {
	EventType  string
	UserID     string
	IPAddress  string
	UserAgent  string
	DeviceInfo string
	Timestamp  time.Time
	Success    bool
	Error      string
	TokenType  string // "access" or "refresh"
	TokenID    string
}

func LogAuthEvent(event AuthEvent) {
	status := "success"
	if !event.Success {
		status = "failure"
	}

	log.Printf("[AUTH] %s | %s | user=%s | ip=%s | device=%s | status=%s | error=%s | token_type=%s | token_id=%s",
		event.EventType,
		event.Timestamp.Format(time.RFC3339),
		event.UserID,
		event.IPAddress,
		event.DeviceInfo,
		status,
		event.Error,
		event.TokenType,
		event.TokenID,
	)
}

func LogTokenRefresh(event AuthEvent) {
	status := "success"
	if !event.Success {
		status = "failure"
	}

	log.Printf("[TOKEN_REFRESH] %s | user=%s | ip=%s | device=%s | status=%s | error=%s | old_token=%s | new_token=%s",
		event.Timestamp.Format(time.RFC3339),
		event.UserID,
		event.IPAddress,
		event.DeviceInfo,
		status,
		event.Error,
		event.TokenID,
		event.TokenType,
	)
}

type AccessEvent struct {
	EventType  string // "permission" or "feature"
	UserID     string
	TenantID   string
	IPAddress  string
	UserAgent  string
	Timestamp  time.Time
	Success    bool
	Error      string
	Resource   string // for permission events
	Action     string // for permission events
	Feature    string // for feature events
}

func LogAccessEvent(event AccessEvent) {
	status := "success"
	if !event.Success {
		status = "failure"
	}

	switch event.EventType {
	case "permission":
		log.Printf("[ACCESS] %s | %s | user=%s | tenant=%s | ip=%s | resource=%s | action=%s | status=%s | error=%s",
			event.EventType,
			event.Timestamp.Format(time.RFC3339),
			event.UserID,
			event.TenantID,
			event.IPAddress,
			event.Resource,
			event.Action,
			status,
			event.Error,
		)
	case "feature":
		log.Printf("[ACCESS] %s | %s | user=%s | tenant=%s | ip=%s | feature=%s | status=%s | error=%s",
			event.EventType,
			event.Timestamp.Format(time.RFC3339),
			event.UserID,
			event.TenantID,
			event.IPAddress,
			event.Feature,
			status,
			event.Error,
		)
	}
}
