package iputils

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ValidateCIDR validates and normalizes a CIDR string
// Returns the normalized CIDR string or an error if invalid
func ValidateCIDR(cidr string) (string, error) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return "", fmt.Errorf("CIDR cannot be empty")
	}

	// Handle single IP addresses by adding appropriate suffix
	if !strings.Contains(cidr, "/") {
		ip := net.ParseIP(cidr)
		if ip == nil {
			return "", fmt.Errorf("invalid IP address: %s", cidr)
		}

		// Add /32 for IPv4 or /128 for IPv6
		if ip.To4() != nil {
			cidr = cidr + "/32"
		} else {
			cidr = cidr + "/128"
		}
	}

	// Parse and validate the CIDR
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR format: %s", err.Error())
	}

	// Return normalized CIDR
	return ipNet.String(), nil
}

// IsIPInCIDRList checks if an IP address is within any of the given CIDR ranges
func IsIPInCIDRList(ipStr string, cidrs []string) (bool, error) {
	clientIP := net.ParseIP(ipStr)
	if clientIP == nil {
		return false, fmt.Errorf("invalid client IP address: %s", ipStr)
	}

	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			// Skip invalid CIDRs but log the error
			continue
		}

		if ipNet.Contains(clientIP) {
			return true, nil
		}
	}

	return false, nil
}

// ExtractClientIP extracts the real client IP from HTTP request
// Handles X-Real-IP and X-Forwarded-For headers for proxy scenarios
// Designed for GCP Load Balancer which appends real client IP to X-Forwarded-For
func ExtractClientIP(r *http.Request) string {
	// Check X-Real-IP header first (set by some GCP configurations and Nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Check X-Forwarded-For header (most common proxy header)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// For GCP Load Balancer, take the rightmost IP (real client IP added by GCP)
		// To prevent spoofing since GCP appends the actual source IP
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[len(ips)-1])
		if clientIP != "" {
			return clientIP
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr might not have port (e.g., in tests)
		return r.RemoteAddr
	}

	return ip
}

// NormalizeIPForStorage normalizes an IP for consistent storage
// Ensures IPv4 addresses are not stored as IPv4-mapped IPv6
func NormalizeIPForStorage(ipStr string) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Convert IPv4-mapped IPv6 addresses back to IPv4
	if ipv4 := ip.To4(); ipv4 != nil {
		return ipv4.String(), nil
	}

	// Return IPv6 address as-is
	return ip.String(), nil
}

// ValidateIPWhitelistRules validates a list of IP/CIDR rules
// Returns normalized rules and any validation errors
func ValidateIPWhitelistRules(rules []string) ([]string, []string) {
	var normalized []string
	var errors []string

	for _, rule := range rules {
		if normalizedRule, err := ValidateCIDR(rule); err != nil {
			errors = append(errors, fmt.Sprintf("Rule '%s': %s", rule, err.Error()))
		} else {
			normalized = append(normalized, normalizedRule)
		}
	}

	return normalized, errors
}
