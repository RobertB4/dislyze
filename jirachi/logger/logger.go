package logger

import (
	"encoding/json"
	"log"
	"time"
)

type AuthEvent struct {
	EventType  string    `json:"event_type"`
	Service    string    `json:"service"`
	UserID     string    `json:"user_id"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	DeviceInfo string    `json:"device_info"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	TokenType  string    `json:"token_type,omitempty"`
	TokenID    string    `json:"token_id,omitempty"`
}

func LogAuthEvent(event AuthEvent) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[AUTH] Failed to marshal auth event: %v", err)
		return
	}
	log.Printf("[AUTH] %s", string(jsonData))
}

func LogTokenRefresh(event AuthEvent) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[TOKEN_REFRESH] Failed to marshal token refresh event: %v", err)
		return
	}
	log.Printf("[TOKEN_REFRESH] %s", string(jsonData))
}

type AccessEvent struct {
	EventType string    `json:"event_type"` // "permission" or "feature"
	Service   string    `json:"service"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	Resource  string    `json:"resource,omitempty"` // for permission events
	Action    string    `json:"action,omitempty"`   // for permission events
	Feature   string    `json:"feature,omitempty"`  // for feature events
}

func LogAccessEvent(event AccessEvent) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[ACCESS] Failed to marshal access event: %v", err)
		return
	}
	log.Printf("[ACCESS] %s", string(jsonData))
}

type RateLimitEvent struct {
	EventType string    `json:"event_type"` // "rate_limit_violation"
	Service   string    `json:"service"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent,omitempty"`
	Endpoint  string    `json:"endpoint"`
	Timestamp time.Time `json:"timestamp"`
	Limit     string    `json:"limit"` // e.g. "5 attempts per hour"
}

func LogRateLimitViolation(event RateLimitEvent) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[RATE_LIMIT] Failed to marshal rate limit event: %v", err)
		return
	}
	log.Printf("[RATE_LIMIT] %s", string(jsonData))
}
