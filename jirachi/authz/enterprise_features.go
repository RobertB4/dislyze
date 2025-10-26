package authz

type EnterpriseFeatures struct {
	RBAC        RBAC        `json:"rbac"`
	IPWhitelist IPWhitelist `json:"ip_whitelist"`
	SSO         SSO         `json:"sso"`
}

type RBAC struct {
	Enabled bool `json:"enabled"`
}

type IPWhitelist struct {
	Enabled                  bool `json:"enabled"` // Internal: Feature available to tenant
	Active                   bool `json:"active"`  // User-controlled: Whether whitelist actively enforces
	AllowInternalAdminBypass bool `json:"allow_internal_admin_bypass,omitempty"`
}

type SSO struct {
	Enabled          bool              `json:"enabled"`
	IdpMetadataURL   string            `json:"idp_metadata_url"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	AllowedDomains   []string          `json:"allowed_domains"`
}
