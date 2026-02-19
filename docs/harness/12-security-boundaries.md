# Workstream #12 — Security Boundaries & Secrets Management

## Current Setup

Secrets are managed at two layers with very different maturity levels:

- **Production**: GCP Secret Manager with KMS encryption, injected into Cloud Run at runtime. No production secrets in the repository. Strong architecture.
- **Local development**: `.env` files committed to git with localhost-only credentials (by design — no production values). One `.env.sensitive` file gitignored for the Anthropic API key. No agent-level access controls.

## What's Already Agent-Friendly (and Secure)

### 1. Production secrets are fully isolated

All production credentials (DB password, JWT secrets, SendGrid key, internal passwords) live in GCP Secret Manager. They're injected into Cloud Run containers via `secretKeyRef` environment bindings. An agent reading the codebase cannot access any production secret.

### 2. No long-lived GCP credentials in the repository

CI/CD uses Workload Identity Federation (OIDC) for GCP authentication. No service account JSON keys exist in the repo, secrets, or environment. This is the current best practice.

### 3. Config loader fails on missing variables

The Go config loader (`lib/config/env.go`) explicitly fails at startup if required environment variables are missing. An agent cannot accidentally deploy code that silently runs without credentials — the service won't start.

### 4. Test credentials are clearly scoped

Integration tests use ephemeral Docker containers with test-only passwords (`test_password`, `1234567890`). These are obviously non-production values. E2E tests use separate credentials (`testpassword_e2e`). The separation is clear.

### 5. Committed `.env` files contain only localhost credentials (by design)

The `.env` files in both backends are intentionally committed. Every value (DB password, JWT secrets, SAML dev key, SendGrid mock key) is localhost-only — none are used in production. Production values live in GCP Secret Manager. This is a deliberate design choice, documented in the root `.gitignore`: "all other .env files are fine, they contain only environment variables for localhost." This is safe and agent-friendly: an agent can read `.env` to understand what configuration the app needs without risking production secret exposure.

## What's NOT Agent-Friendly

### 1. No agent file-access restrictions for `.env.sensitive` (High Impact)

There is no `permissions.deny` configuration in `.claude/settings.json`, no `PreToolUse` hooks, and no file-access restrictions of any kind. The committed `.env` files are safe (localhost-only values), but `.env.sensitive` contains third-party API keys (e.g., Anthropic) that should not be readable by agents.

This was identified in [#2 Development environment](02-dev-environment.md). The API key has been rotated, but any future key placed in `.env.sensitive` is immediately readable.

**Improvement strategy:** Add `permissions.deny` in `.claude/settings.json`:

```json
{
  "permissions": {
    "deny": [
      "Read(./**/*.sensitive)"
    ]
  }
}
```

Place in `.claude/settings.json` (committed, shared with team) for project-wide protection. Files matching these patterns are excluded from Read, Grep, and Glob results. A `PreToolUse` hook can be added later for a custom error message explaining why access was blocked.

### 2. SAML private key not in GCP Secret Manager (Medium Impact)

The SAML private key and certificate are not present in GCP Secret Manager's secret list (per `infrastructure/modules/secrets.ts`). This means either SAML is not enabled in production yet, or the production key is managed through an undocumented channel. Worth clarifying so an agent working on SSO production deployment knows where to find/manage the key.

**Improvement strategy:** When SAML goes to production, add `saml-sp-private-key` and `saml-sp-certificate` to GCP Secret Manager alongside the other production secrets.

### 3. Cross-service JWT secret sharing (Medium Impact)

Giratina holds a copy of lugia's JWT signing secret (`LUGIA_AUTH_JWT_SECRET`). This is needed for giratina to validate lugia session tokens when performing cross-service operations (logging admin users into customer tenants). But it means a compromise of giratina's secret access exposes the ability to forge lugia user tokens.

**Improvement strategy:** Document this coupling in both CLAUDE.md files. Long-term, consider replacing the shared secret with a public-key verification scheme (lugia signs with a private key, giratina verifies with the public key). This limits blast radius.

### 5. No secrets scanning in CI (Medium Impact)

There is no `trufflehog`, `gitleaks`, or similar secrets scanning tool in any CI workflow. If an agent (or human) accidentally commits a real API key, production password, or private key to git, nothing catches it.

**Improvement strategy:** Add `gitleaks` or `trufflehog` as a CI step that runs on every push. Configure it to scan the diff (not full history) for speed. This catches accidental secret commits before they reach the main branch.

### 6. KMS encryption for Secret Manager may not be active (Medium Impact)

The infrastructure code in `infrastructure/modules/secrets.ts` defines a KMS key (`secrets-encryption-key`) for encrypting Secret Manager secrets. But the IAM bindings that grant Secret Manager access to use this KMS key are commented out. This may mean production secrets are using Google-managed encryption (default) rather than customer-managed encryption (CMEK).

**Improvement strategy:** Verify whether the KMS IAM bindings are active in production (check GCP console or `pulumi stack output`). If not, uncomment and deploy. CMEK provides an additional layer of control over secret encryption.

### 7. No documentation of the secrets architecture (Low Impact)

There's no document explaining:
- What secrets exist and why
- Where each secret is stored (local vs. production)
- How to rotate each secret
- What to do if a secret is compromised

An agent working on auth, SSO, or deployment has no guide to the secrets landscape.

**Improvement strategy:** Add a "Secrets Architecture" section to the root CLAUDE.md or create a dedicated `docs/secrets.md`. Cover: what secrets exist, where they live, how they're injected, and the local-vs-production distinction.

### 8. Test credentials are obvious but not documented as test-only (Low Impact)

Passwords like `1234567890`, `password`, `password123`, and `admin123` are committed throughout test infrastructure. These are obviously non-production values, but there's no explicit statement that they must never appear in production configuration.

**Improvement strategy:** Add a comment in the Docker Compose test files: "All credentials in this file are test-only values for ephemeral containers. Production credentials are in GCP Secret Manager."

## Agent Blast Radius Summary

| What an agent can access | Risk if misused |
|---|---|
| `.env.sensitive` (Anthropic API key) | Unauthorized API calls (mitigated: key rotated) |
| `.env` (all values localhost-only) | No production impact — JWT secrets, SAML key, DB password are all local dev values |
| Test passwords in Docker Compose | No production impact |
| GCP production secrets | NOT accessible — properly isolated |

The committed `.env` files are safe by design — they contain only localhost credentials. The only file that needs agent access protection is `.env.sensitive`, which contains third-party API keys that could incur costs or be abused if leaked.

## Summary

| Finding | Impact | Action |
|---|---|---|
| No agent file-access restrictions for `.env.sensitive` | High | Add `permissions.deny` in `.claude/settings.json` |
| SAML key not in GCP Secret Manager | Medium | Add to Secret Manager when SAML goes to production |
| Cross-service JWT secret sharing | Medium | Document coupling, consider public-key verification long-term |
| No secrets scanning in CI | Medium | Add gitleaks/trufflehog to CI |
| KMS encryption may not be active | Medium | Verify and enable CMEK |
| No secrets architecture documentation | Low | Document in CLAUDE.md or docs/secrets.md |
| Test credentials not labeled as test-only | Low | Add comments to Docker Compose files |

The production secrets architecture is sound — GCP Secret Manager, keyless CI auth, no production credentials in the repository. The committed `.env` files are safe by design (localhost-only values). The main gap is agent access to `.env.sensitive` (third-party API keys). Adding `permissions.deny` rules in `.claude/settings.json` closes this gap.
