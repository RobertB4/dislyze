package auth

// AuthConfig defines the configuration interface for authentication middleware
type AuthConfig interface {
	GetJWTSecret() string
	IsCookieSecure() bool
}