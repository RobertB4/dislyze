package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestGenerateTokenPair(t *testing.T) {
	// Test data
	userID := pgtype.UUID{
		Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Valid: true,
	}
	tenantID := pgtype.UUID{
		Bytes: [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		Valid: true,
	}
	role := "admin"
	secret := []byte("test-secret-key")

	tests := []struct {
		name     string
		userID   pgtype.UUID
		tenantID pgtype.UUID
		role     string
		secret   []byte
		wantErr  bool
	}{
		{
			name:     "valid token generation",
			userID:   userID,
			tenantID: tenantID,
			role:     role,
			secret:   secret,
			wantErr:  false,
		},
		{
			name:     "empty secret",
			userID:   userID,
			tenantID: tenantID,
			role:     role,
			secret:   []byte{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPair, err := GenerateTokenPair(tt.userID, tt.tenantID, tt.role, tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tokenPair)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, tokenPair)
			assert.NotEmpty(t, tokenPair.AccessToken)
			assert.NotEmpty(t, tokenPair.RefreshToken)
			assert.Equal(t, int64(15*60), tokenPair.ExpiresIn) // 15 minutes in seconds

			// Validate the access token
			claims, err := ValidateToken(tokenPair.AccessToken, tt.secret)
			assert.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.tenantID, claims.TenantID)
			assert.Equal(t, tt.role, claims.Role)
		})
	}
}

func TestValidateToken(t *testing.T) {
	// Test data
	userID := pgtype.UUID{
		Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Valid: true,
	}
	tenantID := pgtype.UUID{
		Bytes: [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		Valid: true,
	}
	role := "admin"
	secret := []byte("test-secret-key")

	// Generate a valid token
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	validToken, _ := token.SignedString(secret)

	tests := []struct {
		name       string
		token      string
		secret     []byte
		wantErr    bool
		wantClaims *Claims
	}{
		{
			name:       "valid token",
			token:      validToken,
			secret:     secret,
			wantErr:    false,
			wantClaims: &claims,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.string",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			secret:  secret,
			wantErr: true,
		},
		{
			name:    "wrong secret",
			token:   validToken,
			secret:  []byte("wrong-secret"),
			wantErr: true,
		},
		{
			name:    "expired token",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzNDU2Nzg5MCIsInRlbmFudF9pZCI6IjkwODc2NTQzMjEiLCJyb2xlIjoiYWRtaW4iLCJleHAiOjE1MTYyMzkwMjIsImlhdCI6MTUxNjIzOTAyMiwibmJmIjoxNTE2MjM5MDIyfQ.4Adcj3UFYzPUVaVF43FmMze0Qp0j0j6h0h0h0h0h0h0h0",
			secret:  secret,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateToken(tt.token, tt.secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, tt.wantClaims.UserID, claims.UserID)
			assert.Equal(t, tt.wantClaims.TenantID, claims.TenantID)
			assert.Equal(t, tt.wantClaims.Role, claims.Role)
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	userID := pgtype.UUID{
		Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Valid: true,
	}
	tenantID := pgtype.UUID{
		Bytes: [16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1},
		Valid: true,
	}
	secret := []byte("test-secret-key")

	// Generate a token pair
	tokenPair, err := GenerateTokenPair(userID, tenantID, "admin", secret)
	assert.NoError(t, err)
	assert.NotNil(t, tokenPair)

	// Validate the token immediately
	claims, err := ValidateToken(tokenPair.AccessToken, secret)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Check that the expiration time is set correctly
	expectedExp := time.Now().Add(15 * time.Minute)
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
	assert.True(t, claims.ExpiresAt.Time.Before(expectedExp.Add(time.Minute)))
}
