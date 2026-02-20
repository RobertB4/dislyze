// Session data for the harness metrics dashboard.
// Edit this file to add new session scores.
var SESSION_DATA = {
  "rubric": {
    "dimensions": {
      "completion": "Task completion: 1=didn't finish, 3=finished with issues, 5=clean completion",
      "conventions": "Convention adherence: 1=violated patterns, 3=mostly followed, 5=perfectly followed",
      "ci_pass": "First-try CI pass: 1=3+ fix rounds, 3=one fix needed, 5=passed first push",
      "scope": "Scope discipline: 1=significant unrelated changes, 3=minor drift, 5=exactly scoped",
      "self_sufficiency": "Self-sufficiency: 1=constant guidance needed, 3=some correction, 5=fully autonomous"
    },
    "difficulty": "Task difficulty: 1=trivial fix, 2=simple change, 3=multi-file feature, 4=cross-cutting change, 5=architectural change",
    "types": ["bug-fix", "feature", "refactor", "test", "docs", "infra"]
  },
  "sessions": [
    {
      "date": "2025-01-15",
      "task": "Add SSO data structure and basic endpoints",
      "type": "feature",
      "difficulty": 4,
      "harness_version": "baseline",
      "scores": {
        "completion": 4,
        "conventions": 3,
        "ci_pass": 2,
        "scope": 4,
        "self_sufficiency": 2
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "Cross-cutting feature: schema change + backend endpoints + enterprise feature flag. Multiple CI fix commits visible (fix ci partially, try to fix ci)."
    },
    {
      "date": "2025-01-20",
      "task": "Add SSO e2e tests and keycloak integration",
      "type": "test",
      "difficulty": 4,
      "harness_version": "baseline",
      "scores": {
        "completion": 4,
        "conventions": 3,
        "ci_pass": 2,
        "scope": 3,
        "self_sufficiency": 2
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "E2E tests for SSO with keycloak mock. Several fix commits for e2e and CI. Required keycloak environment setup."
    },
    {
      "date": "2025-02-01",
      "task": "Add SSO tenant creation from giratina admin",
      "type": "feature",
      "difficulty": 3,
      "harness_version": "baseline",
      "scores": {
        "completion": 4,
        "conventions": 3,
        "ci_pass": 3,
        "scope": 4,
        "self_sufficiency": 2
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "Add giratina admin support for SSO tenant management. One CI fix visible (fix giratina ci)."
    },
    {
      "date": "2025-02-10",
      "task": "Add IP whitelist to giratina UI",
      "type": "feature",
      "difficulty": 3,
      "harness_version": "baseline",
      "scores": {
        "completion": 4,
        "conventions": 3,
        "ci_pass": 3,
        "scope": 4,
        "self_sufficiency": 2
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "Frontend feature in giratina for IP whitelist management. selfimprove was run after this session but output was too concrete."
    },
    {
      "date": "2025-02-20",
      "task": "Harness audit (workstreams 17-24) + implementation plan + Tier 0 measurement foundation",
      "type": "docs",
      "difficulty": 2,
      "harness_version": "baseline",
      "scores": {
        "completion": 5,
        "conventions": 4,
        "ci_pass": null,
        "scope": 5,
        "self_sufficiency": 4
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "Audited 8 workstreams, created prioritized implementation plan, built Tier 0 (scoring rubric, sessions.js, Chart.js dashboard). CORS bug with fetch on file:// protocol required fix. User provided direction on playwright-cli, abstract vs concrete selfimprove output, baseline score adjustments."
    }
  ]
};
