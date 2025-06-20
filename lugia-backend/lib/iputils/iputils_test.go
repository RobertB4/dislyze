package iputils

import (
	"net/http"
	"testing"
)

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Valid IPv4 CIDR blocks
		{"Valid IPv4 CIDR", "192.168.1.0/24", "192.168.1.0/24", false},
		{"Valid IPv4 single IP", "192.168.1.100", "192.168.1.100/32", false},
		{"Valid IPv4 /32", "10.0.0.1/32", "10.0.0.1/32", false},
		
		// Valid IPv6 CIDR blocks
		{"Valid IPv6 CIDR", "2001:db8::/32", "2001:db8::/32", false},
		{"Valid IPv6 single IP", "2001:db8::1", "2001:db8::1/128", false},
		{"Valid IPv6 /128", "::1/128", "::1/128", false},
		
		// Whitespace handling
		{"IPv4 with whitespace", "  192.168.1.0/24  ", "192.168.1.0/24", false},
		{"Single IP with whitespace", "  10.0.0.1  ", "10.0.0.1/32", false},
		
		// Invalid cases
		{"Empty string", "", "", true},
		{"Invalid IP", "999.999.999.999", "", true},
		{"Invalid CIDR", "192.168.1.0/99", "", true},
		{"Malformed CIDR", "192.168.1.0/", "", true},
		{"Non-IP string", "not-an-ip", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateCIDR(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCIDR() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ValidateCIDR() unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("ValidateCIDR() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsIPInCIDRList(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		cidrs    []string
		expected bool
		wantErr  bool
	}{
		// IPv4 tests
		{"IPv4 in range", "192.168.1.100", []string{"192.168.1.0/24"}, true, false},
		{"IPv4 not in range", "10.0.0.1", []string{"192.168.1.0/24"}, false, false},
		{"IPv4 exact match", "192.168.1.1", []string{"192.168.1.1/32"}, true, false},
		{"IPv4 multiple ranges - match first", "192.168.1.100", []string{"192.168.1.0/24", "10.0.0.0/8"}, true, false},
		{"IPv4 multiple ranges - match second", "10.0.0.100", []string{"192.168.1.0/24", "10.0.0.0/8"}, true, false},
		{"IPv4 multiple ranges - no match", "172.16.0.1", []string{"192.168.1.0/24", "10.0.0.0/8"}, false, false},
		
		// IPv6 tests
		{"IPv6 in range", "2001:db8::100", []string{"2001:db8::/32"}, true, false},
		{"IPv6 not in range", "2001:db9::1", []string{"2001:db8::/32"}, false, false},
		{"IPv6 exact match", "::1", []string{"::1/128"}, true, false},
		
		// Mixed IPv4/IPv6
		{"Mixed ranges - IPv4 match", "192.168.1.100", []string{"2001:db8::/32", "192.168.1.0/24"}, true, false},
		{"Mixed ranges - IPv6 match", "2001:db8::100", []string{"192.168.1.0/24", "2001:db8::/32"}, true, false},
		
		// Error cases
		{"Invalid IP", "999.999.999.999", []string{"192.168.1.0/24"}, false, true},
		{"Empty IP", "", []string{"192.168.1.0/24"}, false, true},
		
		// Edge cases
		{"Empty CIDR list", "192.168.1.100", []string{}, false, false},
		{"Invalid CIDR in list", "192.168.1.100", []string{"invalid-cidr", "192.168.1.0/24"}, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsIPInCIDRList(tt.ip, tt.cidrs)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("IsIPInCIDRList() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("IsIPInCIDRList() unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("IsIPInCIDRList() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name         string
		remoteAddr   string
		headers      map[string]string
		expectedIP   string
	}{
		{
			name:       "Direct connection",
			remoteAddr: "192.168.1.100:12345",
			headers:    map[string]string{},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For single IP",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.100, 10.0.0.2, 10.0.0.3"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "192.168.1.100"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "Both headers - X-Forwarded-For takes precedence",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
				"X-Real-IP":       "192.168.1.200",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For with whitespace",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "  192.168.1.100  , 10.0.0.2"},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.100",
			headers:    map[string]string{},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "IPv6 direct connection",
			remoteAddr: "[2001:db8::1]:12345",
			headers:    map[string]string{},
			expectedIP: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			result := ExtractClientIP(req)
			if result != tt.expectedIP {
				t.Errorf("ExtractClientIP() = %v, want %v", result, tt.expectedIP)
			}
		})
	}
}

func TestNormalizeIPForStorage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"IPv4 address", "192.168.1.100", "192.168.1.100", false},
		{"IPv6 address", "2001:db8::1", "2001:db8::1", false},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.100", "192.168.1.100", false},
		{"Localhost IPv4", "127.0.0.1", "127.0.0.1", false},
		{"Localhost IPv6", "::1", "::1", false},
		{"Invalid IP", "999.999.999.999", "", true},
		{"Empty string", "", "", true},
		{"Non-IP string", "not-an-ip", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeIPForStorage(tt.input)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("NormalizeIPForStorage() expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("NormalizeIPForStorage() unexpected error: %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("NormalizeIPForStorage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateIPWhitelistRules(t *testing.T) {
	tests := []struct {
		name              string
		rules             []string
		expectedNormalized []string
		expectedErrors    int
	}{
		{
			name:              "All valid rules",
			rules:             []string{"192.168.1.0/24", "10.0.0.1", "2001:db8::/32"},
			expectedNormalized: []string{"192.168.1.0/24", "10.0.0.1/32", "2001:db8::/32"},
			expectedErrors:    0,
		},
		{
			name:              "Mixed valid and invalid",
			rules:             []string{"192.168.1.0/24", "invalid-ip", "10.0.0.1"},
			expectedNormalized: []string{"192.168.1.0/24", "10.0.0.1/32"},
			expectedErrors:    1,
		},
		{
			name:              "All invalid rules",
			rules:             []string{"invalid-ip", "999.999.999.999", "bad/cidr"},
			expectedNormalized: []string{},
			expectedErrors:    3,
		},
		{
			name:              "Empty rule list",
			rules:             []string{},
			expectedNormalized: []string{},
			expectedErrors:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, errors := ValidateIPWhitelistRules(tt.rules)
			
			if len(normalized) != len(tt.expectedNormalized) {
				t.Errorf("ValidateIPWhitelistRules() normalized count = %v, want %v", 
					len(normalized), len(tt.expectedNormalized))
			}
			
			for i, expected := range tt.expectedNormalized {
				if i < len(normalized) && normalized[i] != expected {
					t.Errorf("ValidateIPWhitelistRules() normalized[%d] = %v, want %v", 
						i, normalized[i], expected)
				}
			}
			
			if len(errors) != tt.expectedErrors {
				t.Errorf("ValidateIPWhitelistRules() error count = %v, want %v", 
					len(errors), tt.expectedErrors)
			}
		})
	}
}