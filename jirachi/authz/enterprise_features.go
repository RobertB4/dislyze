package authz

type EnterpriseFeatures struct {
	RBAC RBAC `json:"rbac"`
}

type RBAC struct {
	Enabled bool `json:"enabled"`
}
