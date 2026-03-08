// Feature doc: docs/features/audit-logging.md
package auditlog

// ResourceType identifies the category of audited entity.
// Uses singular form to distinguish from RBAC permissions (which use plural).
type ResourceType string

const (
	ResourceAuth        ResourceType = "auth"
	ResourceAccess      ResourceType = "access"
	ResourceUser        ResourceType = "user"
	ResourceRole        ResourceType = "role"
	ResourceIPWhitelist ResourceType = "ip_whitelist"
	ResourceTenant      ResourceType = "tenant"
)

// Action identifies the specific operation within a resource type.
// Uses past-tense verbs to distinguish from RBAC actions (which use present-tense capabilities).
type Action string

// Auth actions
const (
	ActionLogin                  Action = "login"
	ActionLogout                 Action = "logout"
	ActionPasswordChanged        Action = "password_changed"
	ActionPasswordResetRequested Action = "password_reset_requested"
	ActionPasswordResetCompleted Action = "password_reset_completed"
)

// Access actions (always outcome: failure)
const (
	ActionPermissionDenied  Action = "permission_denied"
	ActionFeatureGateBlocked Action = "feature_gate_blocked"
	ActionIPBlocked         Action = "ip_blocked"
)

// User management actions
const (
	ActionInvited      Action = "invited"
	ActionDeleted      Action = "deleted"
	ActionEmailChanged Action = "email_changed"
	ActionRolesUpdated Action = "roles_updated"
	ActionInviteResent Action = "invite_resent"
	ActionListViewed   Action = "list_viewed"
)

// Role management actions
const (
	ActionCreated Action = "created"
	ActionUpdated Action = "updated"
	// ActionDeleted is reused from user management
)

// IP whitelist actions
const (
	ActionActivated             Action = "activated"
	ActionDeactivated           Action = "deactivated"
	ActionIPAdded               Action = "ip_added"
	ActionIPRemoved             Action = "ip_removed"
	ActionIPUpdated             Action = "ip_updated"
	ActionEmergencyDeactivated  Action = "emergency_deactivated"
)

// Tenant management actions
const (
	ActionNameChanged             Action = "name_changed"
	ActionEnterpriseFeatureToggled Action = "enterprise_feature_toggled"
)

// Outcome represents the result of an audited action.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
)
