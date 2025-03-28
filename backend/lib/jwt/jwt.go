package jwt

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Claims struct {
	UserID   pgtype.UUID `json:"user_id"`
	TenantID pgtype.UUID `json:"tenant_id"`
	Role     string      `json:"role"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// GenerateTokenPair creates a new access token and refresh token
func GenerateTokenPair(userID, tenantID pgtype.UUID, role string, secret []byte) (*TokenPair, error) {
	// Generate a random refresh token
	refreshToken := make([]byte, 32)
	if _, err := rand.Read(refreshToken); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenString := base64.URLEncoding.EncodeToString(refreshToken)

	// Create access token claims
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)), // Access token expires in 15 minutes
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create access token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		ExpiresIn:    15 * 60, // 15 minutes in seconds
	}, nil
}

// ValidateAccessToken validates an access token and returns its claims
func ValidateAccessToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
