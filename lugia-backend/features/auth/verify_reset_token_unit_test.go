package auth

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyResetTokenRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request VerifyResetTokenRequestBody
		wantErr error
	}{
		{
			name: "valid request",
			request: VerifyResetTokenRequestBody{
				Token: "valid-token-123",
			},
			wantErr: nil,
		},
		{
			name: "missing token",
			request: VerifyResetTokenRequestBody{
				Token: "",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "empty token",
			request: VerifyResetTokenRequestBody{
				Token: "",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "whitespace-only token",
			request: VerifyResetTokenRequestBody{
				Token: "   ",
			},
			wantErr: fmt.Errorf("token is required"),
		},
		{
			name: "token with leading/trailing whitespace",
			request: VerifyResetTokenRequestBody{
				Token: "  valid-token-123  ",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
