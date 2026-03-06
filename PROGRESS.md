# Progress

## Completed

- Agent guardrails: PostToolUse hooks (dependency-awareness, duplication-radar, pattern-guide)
- Agent guardrails: UserPromptSubmit hook (principle injection with keyword triggers)
- ESLint func-style rule + all violations converted
- Prettier/ignore fixes across all frontends

## In Flight

### Autonomous Agents (local-first, subscription-only)

Goal: Build agent workflows that run locally using Claude Code subscription, prove value, then automate in CI later.

---

#### Track A: Roaming Test Agent

An agent that explores the codebase, finds undertested areas, writes tests, runs them, self-reviews, and creates PRs.

**Constraints:**
- No mocks (project rule) — unit tests for pure functions, integration tests with real deps, E2E with Playwright
- Tests must verify behavior, not implementation
- Agent must run tests and confirm they pass before PR
- A separate reviewer pass (different prompt) should validate test quality

**Steps:**

1. **Design the test agent prompt**
   - System prompt that encodes testing principles, codebase structure, and what "good test" means
   - Must reference CLAUDE.md and testing conventions
   - Should instruct agent to: pick an area, read existing tests, identify gaps, write tests, run them, self-review
   - Status: TODO

2. **Design the reviewer prompt**
   - Separate prompt with adversarial stance: "find tests that are tautological, trivial, or don't test real behavior"
   - Should check: does the test break if the implementation changes? Does it cover edge cases?
   - Status: TODO

3. **Create a runner script**
   - Shell script that orchestrates: test agent writes tests → runs `make verify` → reviewer agent reviews → if approved, stages PR
   - Uses `claude -p` (headless mode) with subscription auth
   - Status: TODO

4. **Dry run on one module**
   - Pick a module (e.g., jirachi — shared library, pure functions, easiest to test)
   - Run the agent, review output quality manually
   - Iterate on prompts based on results
   - Status: TODO

5. **Expand to other modules**
   - Once prompt quality is proven, run on backends and frontends
   - Status: TODO

6. **Automate (future)**
   - Move to GitHub Actions with API key when ready
   - Weekly cron schedule
   - Status: FUTURE

---

#### Track B: Code Review Agent

Specialized agents that review PRs against project conventions, architecture, and security rules.

**Constraints:**
- Must have full codebase context (not just diff) to be useful
- High-confidence findings only — noise kills adoption
- Single well-prompted agent first, specialize later
- Should leverage existing CLAUDE.md files as review criteria

**Steps:**

1. **Design the review agent prompt**
   - Focus: conformance to CLAUDE.md conventions, architecture violations, pattern consistency
   - Must distinguish severity: "this will break" vs "this could be improved"
   - Only post high-confidence findings
   - Status: TODO

2. **Create a runner script**
   - Takes a branch name or diff as input
   - Runs `claude -p` with the review prompt + diff context
   - Outputs structured findings (file, line, severity, message)
   - Status: TODO

3. **Test on recent PRs**
   - Run against the last 3-5 merged PRs
   - Compare agent findings to actual issues found (or missed) in human review
   - Iterate on prompt based on false positives/negatives
   - Status: TODO

4. **Integrate into workflow**
   - Run before merging PRs (locally or as a pre-merge check)
   - Status: TODO

5. **Specialize (future)**
   - If single agent proves valuable, split into specialized agents (security, architecture, consistency)
   - Add severity filtering and aggregation
   - Status: FUTURE

---

## Decisions

- **Local-first**: All agent workflows use `claude -p` with subscription auth. No API key needed initially.
- **Prove before automating**: Get prompt quality right locally before moving to CI.
- **Single reviewer > multiple specialists**: Start with one well-prompted review agent.
- **Test agent needs separate reviewer**: Writer and reviewer must be different agent invocations with different prompts.
