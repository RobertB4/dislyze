package jwt

import (
	"errors"
	"fmt"
	"time"

	"dislyze/lib/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrTokenExpired          = errors.New("token is expired")
	ErrTokenNotValidYet      = errors.New("token is not valid yet")
	ErrTokenMalformed        = errors.New("token is malformed")
	ErrTokenInvalidSignature = errors.New("token signature is invalid")
	ErrTokenUsedBeforeIssued = errors.New("token used before issued")
	ErrTokenInvalid          = errors.New("token is invalid for other reasons")
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

func GenerateAccessToken(userID, tenantID pgtype.UUID, role string, secret []byte) (string, int64, *Claims, error) {
	if len(secret) == 0 {
		return "", 0, nil, fmt.Errorf("secret cannot be empty")
	}

	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)), // Access token expires in 15 minutes
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(secret)
	if err != nil {
		return "", 0, nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return accessToken, 15 * 60, claims, nil // 15 minutes in seconds, return claims
}

func GenerateRefreshToken(userID pgtype.UUID, secret []byte) (string, pgtype.UUID, error) {
	if len(secret) == 0 {
		return "", pgtype.UUID{}, fmt.Errorf("secret cannot be empty")
	}

	jti, err := utils.NewUUID()
	if err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to generate jti for refresh token: %w", err)
	}

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

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	refreshToken, err := token.SignedString(secret)
	if err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return refreshToken, jti, nil
}

func GenerateTokenPair(userID, tenantID pgtype.UUID, role string, secret []byte) (*TokenPair, error) {
	accessToken, expiresIn, _, err := GenerateAccessToken(userID, tenantID, role, secret)
	if err != nil {
		return nil, err
	}

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

func ValidateToken(tokenString string, secret []byte) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		} else if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		} else if errors.Is(err, jwt.ErrTokenNotValidYet) {
			return nil, ErrTokenNotValidYet
		} else if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrTokenInvalidSignature
		} else if errors.Is(err, jwt.ErrTokenUsedBeforeIssued) {
			return nil, ErrTokenUsedBeforeIssued
		}
		// Fallback for other parsing errors or validation errors not specifically handled above.
		// We wrap the original error to preserve its context.
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	// This case should ideally not be reached if token.Valid is false and err was nil,
	// but as a fallback, we return a generic invalid token error.
	return nil, ErrTokenInvalid
}
