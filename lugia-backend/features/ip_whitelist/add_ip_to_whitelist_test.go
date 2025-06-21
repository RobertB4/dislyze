package ip_whitelist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddIPToWhitelistRequest_Validate(t *testing.T) {
	tests := []struct {
		name          string
		request       AddIPToWhitelistRequest
		expectedError bool
		errorContains string
	}{
		{
			name: "valid IPv4 address",
			request: AddIPToWhitelistRequest{
				IPAddress: "192.168.1.100",
				Label:     stringPtr("Office IP"),
			},
			expectedError: false,
		},
		{
			name: "valid IPv4 CIDR range",
			request: AddIPToWhitelistRequest{
				IPAddress: "172.18.0.0/16",
				Label:     stringPtr("Docker Network"),
			},
			expectedError: false,
		},
		{
			name: "valid IPv6 address",
			request: AddIPToWhitelistRequest{
				IPAddress: "2001:db8::1",
				Label:     stringPtr("IPv6 Address"),
			},
			expectedError: false,
		},
		{
			name: "valid IPv6 CIDR range",
			request: AddIPToWhitelistRequest{
				IPAddress: "2001:db8::/64",
				Label:     stringPtr("IPv6 Subnet"),
			},
			expectedError: false,
		},
		{
			name: "empty IP address",
			request: AddIPToWhitelistRequest{
				IPAddress: "",
				Label:     stringPtr("Empty IP"),
			},
			expectedError: true,
			errorContains: "",
		},
		{
			name: "invalid IP address",
			request: AddIPToWhitelistRequest{
				IPAddress: "999.999.999.999",
				Label:     stringPtr("Invalid IP"),
			},
			expectedError: true,
			errorContains: "",
		},
		{
			name: "invalid CIDR notation",
			request: AddIPToWhitelistRequest{
				IPAddress: "192.168.1.0/99",
				Label:     stringPtr("Invalid CIDR"),
			},
			expectedError: true,
			errorContains: "",
		},
		{
			name: "malformed input",
			request: AddIPToWhitelistRequest{
				IPAddress: "not-an-ip-address",
				Label:     stringPtr("Malformed"),
			},
			expectedError: true,
			errorContains: "",
		},
		{
			name: "valid without label",
			request: AddIPToWhitelistRequest{
				IPAddress: "10.0.0.1",
				Label:     nil,
			},
			expectedError: false,
		},
		{
			name: "IPv6 localhost",
			request: AddIPToWhitelistRequest{
				IPAddress: "::1",
				Label:     stringPtr("IPv6 localhost"),
			},
			expectedError: false,
		},
		{
			name: "broad IPv4 range",
			request: AddIPToWhitelistRequest{
				IPAddress: "0.0.0.0/0",
				Label:     stringPtr("All IPv4"),
			},
			expectedError: false,
		},
		{
			name: "broad IPv6 range",
			request: AddIPToWhitelistRequest{
				IPAddress: "::/0",
				Label:     stringPtr("All IPv6"),
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
