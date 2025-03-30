package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
	JWTSecret   string
	FrontendURL string
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
		"DB_HOST":      &env.DBHost,
		"DB_PORT":      &env.DBPort,
		"DB_USER":      &env.DBUser,
		"DB_PASSWORD":  &env.DBPassword,
		"DB_NAME":      &env.DBName,
		"DB_SSL_MODE":  &env.DBSSLMode,
		"JWT_SECRET":   &env.JWTSecret,
		"FRONTEND_URL": &env.FrontendURL,
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
