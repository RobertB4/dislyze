package authz

type EnterpriseFeatures struct {
	RBAC        RBAC        `json:"rbac"`
	IPWhitelist IPWhitelist `json:"ip_whitelist"`
	SSO         SSO         `json:"sso,omitempty"`
	AuditLog    AuditLog    `json:"audit_log"`
}

type RBAC struct {
	Enabled bool `json:"enabled"`
}

type IPWhitelist struct {
	Enabled                  bool `json:"enabled"`                  // Internal: Feature available to tenant
	Active                   bool `json:"active"`                   // User-controlled: Whether whitelist actively enforces
	AllowInternalAdminBypass bool `json:"allow_internal_admin_bypass"`
}

type AuditLog struct {
	Enabled bool `json:"enabled"`
}

type SSO struct {
	Enabled          bool              `json:"enabled"`
	IdpMetadataURL   string            `json:"idp_metadata_url,omitempty"`
	AttributeMapping map[string]string `json:"attribute_mapping,omitempty"`
	AllowedDomains   []string          `json:"allowed_domains,omitempty"`
}
