package auth

type AuthConfig interface {
	GetJWTSecret() string
	IsCookieSecure() bool
}
