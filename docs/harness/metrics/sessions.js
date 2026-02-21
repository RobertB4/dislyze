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
    },
    {
      "date": "2025-02-20",
      "task": "Tier 1 harness implementation (15 quick wins) + CI fixes",
      "type": "infra",
      "difficulty": 3,
      "harness_version": "tier-1",
      "scores": {
        "completion": 5,
        "conventions": 4,
        "ci_pass": 3,
        "scope": 4,
        "self_sufficiency": 3
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "Implemented all 15 Tier 1 items: Makefile verify/generate targets, CI pull_request triggers, PR template, .claude/settings.json deny rules, /selfimprove rewrite, /review command, CLAUDE.md knowledge layer (architecture, DoD, escalation, blast radius), jirachi/zoroark CLAUDE.md. CI fix pass: Go 1.24.13 bump (govulncheck), eslint 9->10 migration (npm audit), gosec #nosec annotations + test dir exclusion, svelte-check type imports + adapter-node dep. User corrections: (1) inconsistent jirachi lint in Makefile, (2) missing zoroark check, (3) npm override proposed before upgrading direct deps (led to 'Root cause over workaround' principle), (4) reminded to verify changes work after making them."
    },
    {
      "date": "2026-02-21",
      "task": "Tier 2 structural tests + command improvements + CI trigger fix",
      "type": "infra",
      "difficulty": 3,
      "harness_version": "tier-2",
      "scores": {
        "completion": 4,
        "conventions": 3,
        "ci_pass": 4,
        "scope": 4,
        "self_sufficiency": 2
      },
      "turns": null,
      "duration_minutes": null,
      "notes": "Structural tests (verify-generated-code, verify-claude-refs), /peer-review, /bigpicture commands, /review process checklist, CI push triggers restricted to main. User corrections: (1) PR description didn't follow template, (2) peer-review had unnecessary usage line, (3) peer-review should allow autonomous submission, (4) peer-review needed explore step, (5) feature list placement — pushed to think about when info is needed not just where, (6) CI trigger explanation was wrong twice (said push only on main, actually all branches), (7) duplicate CI runs not caught by agent, (8) review checklist 'PR ready' marked checked with no PR, (9) implemented wrong selfimprove items. Learnings: verify before explaining, honest checklist evaluation, information timing."
    }
  ]
};
