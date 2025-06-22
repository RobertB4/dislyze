package jwt

import (
	"fmt"
	"time"

	"dislyze/jirachi/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type EmergencyClaims struct {
	UserID   pgtype.UUID `json:"user_id"`
	TenantID pgtype.UUID `json:"tenant_id"`
	Action   string      `json:"action"`
	JTI      pgtype.UUID `json:"jti"`
	jwt.RegisteredClaims
}

func GenerateEmergencyToken(userID, tenantID pgtype.UUID, secret []byte) (string, pgtype.UUID, error) {
	if len(secret) == 0 {
		return "", pgtype.UUID{}, fmt.Errorf("secret cannot be empty")
	}

	jti, err := utils.NewUUID()
	if err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to generate jti for emergency token: %w", err)
	}

	now := time.Now()
	claims := EmergencyClaims{
		UserID:   userID,
		TenantID: tenantID,
		Action:   "ip_whitelist.emergency_deactivate",
		JTI:      jti,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", pgtype.UUID{}, fmt.Errorf("failed to sign emergency token: %w", err)
	}

	return tokenString, jti, nil
}

func ValidateEmergencyToken(tokenString string, secret []byte) (*EmergencyClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &EmergencyClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid emergency token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("emergency token signature is invalid")
	}

	claims, ok := token.Claims.(*EmergencyClaims)
	if !ok {
		return nil, fmt.Errorf("emergency token claims are invalid")
	}

	if claims.Action != "ip_whitelist.emergency_deactivate" {
		return nil, fmt.Errorf("emergency token has invalid action: %s", claims.Action)
	}

	if claims.ExpiresAt == nil {
		return nil, fmt.Errorf("emergency token has no ExpiresAt set")
	}

	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("emergency token has expired")
	}

	return claims, nil
}
