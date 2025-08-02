package auth

type AuthConfig interface {
	GetAuthJWTSecret() string
	IsCookieSecure() bool
}
