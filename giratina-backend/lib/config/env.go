package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// GiratinaAuthConfig implements jirachi's AuthConfig interface
type GiratinaAuthConfig struct {
	env *Env
}

func NewGiratinaAuthConfig(env *Env) *GiratinaAuthConfig {
	return &GiratinaAuthConfig{env: env}
}

func (c *GiratinaAuthConfig) GetJWTSecret() string {
	return c.env.JWTSecret
}

func (c *GiratinaAuthConfig) IsCookieSecure() bool {
	return c.env.IsCookieSecure()
}

type Env struct {
	AppEnv                string
	Port                  string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DBSSLMode             string
	JWTSecret             string
	AuthRateLimit         string
	CreateTenantJwtSecret string
	FrontendURL           string
	LugiaFrontendUrl      string
}

func LoadEnv() (*Env, error) {
	// Only load .env file in local development
	if os.Getenv("APP_ENV") == "" || os.Getenv("APP_ENV") == "local" {
		if err := godotenv.Load(".env"); err != nil {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	env := &Env{}

	required := map[string]*string{
		"APP_ENV":                  &env.AppEnv,
		"PORT":                     &env.Port,
		"DB_HOST":                  &env.DBHost,
		"DB_PORT":                  &env.DBPort,
		"DB_USER":                  &env.DBUser,
		"DB_PASSWORD":              &env.DBPassword,
		"DB_NAME":                  &env.DBName,
		"DB_SSL_MODE":              &env.DBSSLMode,
		"JWT_SECRET":               &env.JWTSecret,
		"AUTH_RATE_LIMIT":          &env.AuthRateLimit,
		"CREATE_TENANT_JWT_SECRET": &env.CreateTenantJwtSecret,
		"FRONTEND_URL":             &env.FrontendURL,
		"LUGIA_FRONTEND_URL":       &env.LugiaFrontendUrl,
	}

	var missing []string
	for key, value := range required {
		if *value = os.Getenv(key); *value == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return env, nil
}

// Returns false for local and e2e environments, true for all others
func (e *Env) IsCookieSecure() bool {
	return e.AppEnv != "local" && e.AppEnv != "e2e"
}
