package jwt

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Claims struct {
	UserID   pgtype.UUID `json:"user_id"`
	TenantID pgtype.UUID `json:"tenant_id"`
	Role     string      `json:"role"`
	JTI      pgtype.UUID `json:"jti"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	JTI          pgtype.UUID
}

// GenerateAccessToken creates a new access token
func GenerateAccessToken(userID, tenantID pgtype.UUID, role string, secret []byte) (string, int64, error) {
	// Validate secret
	if len(secret) == 0 {
		return "", 0, fmt.Errorf("secret cannot be empty")
	}

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
		return "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	return accessToken, 15 * 60, nil // 15 minutes in seconds
}

// GenerateRefreshToken creates a new refresh token
func GenerateRefreshToken(userID pgtype.UUID, secret []byte) (string, pgtype.UUID, error) {
	// Validate secret
	if len(secret) == 0 {
		return "", pgtype.UUID{}, fmt.Errorf("secret cannot be empty")
	}

	// Generate random bytes for the token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Generate a new UUID for the jti claim
	var jti pgtype.UUID
	jti.Bytes = [16]byte{}
	if _, err := rand.Read(jti.Bytes[:]); err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to generate jti: %w", err)
	}
	jti.Valid = true

	// Create refresh token claims
	now := time.Now()
	claims := Claims{
		UserID: userID,
		JTI:    jti,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)), // Refresh token expires in 7 days
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create refresh token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshToken, err := token.SignedString(secret)
	if err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return refreshToken, jti, nil
}

// GenerateTokenPair creates a new access token and refresh token
func GenerateTokenPair(userID, tenantID pgtype.UUID, role string, secret []byte) (*TokenPair, error) {
	// Generate access token
	accessToken, expiresIn, err := GenerateAccessToken(userID, tenantID, role, secret)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, jti, err := GenerateRefreshToken(userID, secret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		JTI:          jti,
	}, nil
}

// ValidateToken validates a token and returns its claims
func ValidateToken(tokenString string, secret []byte) (*Claims, error) {
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
