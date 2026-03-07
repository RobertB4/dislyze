package authz

type EnterpriseFeatures struct {
	RBAC        RBAC        `json:"rbac,omitempty"`
	IPWhitelist IPWhitelist `json:"ip_whitelist,omitempty"`
	SSO         SSO         `json:"sso,omitempty"`
}

type RBAC struct {
	Enabled bool `json:"enabled,omitempty"`
}

type IPWhitelist struct {
	Enabled                  bool `json:"enabled,omitempty"`                       // Internal: Feature available to tenant
	Active                   bool `json:"active,omitempty"`                        // User-controlled: Whether whitelist actively enforces
	AllowInternalAdminBypass bool `json:"allow_internal_admin_bypass,omitempty"`
}

type SSO struct {
	Enabled          bool              `json:"enabled,omitempty"`
	IdpMetadataURL   string            `json:"idp_metadata_url,omitempty"`
	AttributeMapping map[string]string `json:"attribute_mapping,omitempty"`
	AllowedDomains   []string          `json:"allowed_domains,omitempty"`
}
