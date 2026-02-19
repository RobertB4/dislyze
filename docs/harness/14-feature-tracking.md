# Workstream #14 — Feature Tracking System

## Current Setup

There is no feature tracking system. No `feature-list.json`, no issue templates, no CHANGELOG, no git tags, no ROADMAP, no TODO.md, and zero TODO/FIXME comments in the codebase.

The only feature enumeration that exists is implicit: enterprise features (RBAC, IP Whitelist, SSO) are tracked through typed Go constants spread across 4 files. Application-level features (auth, invitations, password reset, email change, etc.) are not enumerated anywhere — they exist only as code.

## What's Already Agent-Friendly

### 1. The codebase is clean of scattered TODOs

Zero `TODO`, `FIXME`, `HACK`, or `XXX` comments exist anywhere. This means agents don't encounter orphaned work items mixed into the code. There's no "TODO debt" to confuse or distract.

### 2. Enterprise features are typed constants

The three enterprise features are defined as Go constants with a typed `EnterpriseFeature` string type. An agent can enumerate them by reading `lugia/lib/authz/enterprise_features.go`. The `lugia-backend/CLAUDE.md` documents the multi-file addition pattern step-by-step.

### 3. HARNESS.md provides process-level tracking

The harness engineering workstream table in `HARNESS.md` tracks what needs to be done at the environment/tooling level. It's not feature tracking, but it demonstrates the project can maintain a structured tracking document.

## What's NOT Agent-Friendly

### 1. No machine-readable feature inventory (High Impact)

An agent asked "add a new page for feature X" has no way to look up what features exist, what state they're in, or what their boundaries are. The agent must explore the codebase — read route files, handler registrations, and database tables — to build a mental model of what the application does.

In a harness engineering environment, agents should be able to read a single file to understand the feature landscape. This is what `feature-list.json` is designed to solve.

**Improvement strategy:** Create `feature-list.json` at the repository root. Structure it as a machine-readable inventory:

```json
{
  "features": [
    {
      "id": "auth",
      "name": "Authentication",
      "description": "JWT auth with refresh tokens, login, logout, signup, password reset, email change",
      "status": "implemented",
      "backend": "lugia-backend/features/auth/",
      "frontend": "lugia-frontend/src/routes/auth/",
      "enterprise": false
    },
    {
      "id": "rbac",
      "name": "Role-Based Access Control",
      "description": "Custom roles, permissions, role assignment per tenant",
      "status": "implemented",
      "backend": "lugia-backend/features/roles/",
      "frontend": "lugia-frontend/src/routes/settings/roles/",
      "enterprise": true
    }
  ]
}
```

This gives agents: what features exist, where they live in the codebase, whether they're enterprise-gated, and what their status is. The file is both human-readable and machine-parseable — a structural test could validate that every feature's paths actually exist.

### 2. No GitHub issue templates or PR templates (Medium Impact)

There are no issue templates, no PR template, and no `CODEOWNERS` file. When agents create PRs (or when the review pipeline from workstream #19 is implemented), there's no structured format for describing changes.

**Improvement strategy:** Add at minimum:
- A PR template (`.github/PULL_REQUEST_TEMPLATE.md`) that prompts for: what changed, why, what was tested, and which feature this relates to (referencing `feature-list.json` IDs)
- A `CODEOWNERS` file mapping directories to reviewers (even if it's just one person for now — it documents ownership)

Issue templates can wait until there's a multi-agent or multi-developer workflow.

### 3. No changelog or version tracking (Low Impact)

No `CHANGELOG.md`, no git tags, no semantic versioning on any package. The app is pre-production, so this is understandable — there's nothing to "release" yet. But as the app moves toward production, a changelog becomes important for agents to understand what changed between versions.

**Improvement strategy:** Low priority for now. When the app approaches production, add a CHANGELOG.md (keep-a-changelog format) and start tagging releases. Consider automating changelog generation from conventional commits (which ties into workstream #15 — version control strategy).

### 4. giratina-backend "Next Steps" is stale scaffold text (Low Impact)

`giratina-backend/CLAUDE.md` lists "Authentication middleware, Error handling utilities, Response formatting utilities, Logging, Rate limiting" as things to be added — but several of these already exist (via jirachi). The section is leftover scaffold text that gives agents a false picture of giratina's completeness.

**Improvement strategy:** Remove or rewrite the "Next Steps" section. Either list actual planned work or remove it entirely and let the feature tracking system handle forward-looking information.

## What `feature-list.json` Should Capture

Based on the current codebase, the initial feature inventory would be:

| Feature | Status | Enterprise | Backend | Frontend |
|---|---|---|---|---|
| Authentication (login/logout/signup) | Implemented | No | `lugia-backend/features/auth/` | `lugia-frontend/src/routes/auth/` |
| JWT + Refresh Token Lifecycle | Implemented | No | `jirachi/auth/`, `jirachi/jwt/` | — |
| Password Reset | Implemented | No | `lugia-backend/features/auth/` | `lugia-frontend/src/routes/auth/` |
| Email Change | Implemented | No | `lugia-backend/features/users/` | `lugia-frontend/src/routes/verify/` |
| User Invitations | Implemented | No | `lugia-backend/features/users/` | `lugia-frontend/src/routes/auth/accept-invite/` |
| User Management | Implemented | No | `lugia-backend/features/users/` | `lugia-frontend/src/routes/settings/users/` |
| RBAC (Roles & Permissions) | Implemented | Yes | `lugia-backend/features/roles/` | `lugia-frontend/src/routes/settings/roles/` |
| IP Whitelist | Implemented | Yes | `lugia-backend/features/ip_whitelist/` | `lugia-frontend/src/routes/settings/ip-whitelist/` |
| SSO (SAML/Keycloak) | Implemented | Yes | `lugia-backend/features/auth/sso_*` | `lugia-frontend/src/routes/auth/sso-login/` |
| Tenant Administration | Implemented | No | `giratina-backend/features/tenants/` | `giratina-frontend/src/routes/` |
| Profile Settings | Implemented | No | `lugia-backend/features/users/` | `lugia-frontend/src/routes/settings/profile/` |

## Summary

| Finding | Impact | Action |
|---|---|---|
| No machine-readable feature inventory | High | Create `feature-list.json` with feature IDs, paths, status |
| No GitHub issue/PR templates | Medium | Add PR template, CODEOWNERS |
| No changelog or version tracking | Low | Add when approaching production |
| Stale "Next Steps" in giratina CLAUDE.md | Low | Remove or rewrite |

The feature tracking system is the simplest workstream to implement — it's a single JSON file. But its impact is outsized: it gives agents a map of the application's capabilities, links features to code locations, and provides the vocabulary for referencing features in PRs, commits, and documentation. Combined with structural tests that validate the file against actual code paths, it becomes a self-maintaining inventory.
