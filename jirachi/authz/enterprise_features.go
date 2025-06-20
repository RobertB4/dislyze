package authz

type EnterpriseFeatures struct {
	RBAC        RBAC        `json:"rbac"`
	IPWhitelist IPWhitelist `json:"ip_whitelist"`
}

type RBAC struct {
	Enabled bool `json:"enabled"`
}

type IPWhitelist struct {
	Enabled                   bool `json:"enabled"`
	AllowInternalAdminBypass bool `json:"allow_internal_admin_bypass,omitempty"`
}
