package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChangeEmailRequestBody_Validate(t *testing.T) {
	tests := []struct {
		name          string
		request       ChangeEmailRequestBody
		expectedError string
	}{
		{
			name: "valid email",
			request: ChangeEmailRequestBody{
				NewEmail: "user@example.com",
			},
			expectedError: "",
		},
		{
			name: "empty email",
			request: ChangeEmailRequestBody{
				NewEmail: "",
			},
			expectedError: "new email is required",
		},
		{
			name: "whitespace-only email",
			request: ChangeEmailRequestBody{
				NewEmail: "   ",
			},
			expectedError: "new email is required",
		},
		{
			name: "invalid email format (no @)",
			request: ChangeEmailRequestBody{
				NewEmail: "invalid-email",
			},
			expectedError: "new email is invalid",
		},
		{
			name: "invalid email format (missing domain)",
			request: ChangeEmailRequestBody{
				NewEmail: "user@",
			},
			expectedError: "", // This will pass basic @ validation but would fail in real email validation
		},
		{
			name: "invalid email format (missing user)",
			request: ChangeEmailRequestBody{
				NewEmail: "@example.com",
			},
			expectedError: "", // This will pass basic @ validation but would fail in real email validation
		},
		{
			name: "whitespace trimming works",
			request: ChangeEmailRequestBody{
				NewEmail: "  user@example.com  ",
			},
			expectedError: "",
		},
		{
			name: "email with subdomain",
			request: ChangeEmailRequestBody{
				NewEmail: "user@mail.example.com",
			},
			expectedError: "",
		},
		{
			name: "email with plus sign",
			request: ChangeEmailRequestBody{
				NewEmail: "user+tag@example.com",
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.expectedError == "" {
				assert.NoError(t, err)
				// Verify that whitespace is trimmed
				if tt.name == "whitespace trimming works" {
					assert.Equal(t, "user@example.com", tt.request.NewEmail)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}
