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
	Provider         string            `json:"provider"`
	IdpURL           string            `json:"idp_url"`
	IdpCertificate   string            `json:"idp_certificate"`
	EntityID         string            `json:"entity_id"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	AllowedDomains   []string          `json:"allowed_domains"`
}
