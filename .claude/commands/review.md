Review the current diff before committing. This is a self-review step — catch problems before they reach CI or human review.

## Steps

### 1. Read the diff

Run `git diff` (and `git diff --cached` if anything is staged) to see all changes.

### 2. Code review

For every changed file, check:

- [ ] **Scope**: Does this change relate to the task? Flag any unrelated modifications.
- [ ] **Generated code**: Are changes to `queries/` directories made via `make sqlc`, not hand-edited?
- [ ] **Conventions**: Does the code follow the patterns in the relevant CLAUDE.md?
- [ ] **Error handling**: Are errors handled consistently with the rest of the codebase?
- [ ] **Security**: No secrets, no SQL injection, no XSS, no command injection?
- [ ] **Tests**: If behavior changed, are tests updated?
- [ ] **Comments**: Comments explain WHY, not WHAT. No unnecessary comments added.
- [ ] **Anything else**: If something feels off but isn't covered above, flag it. This checklist is not exhaustive — use your judgement.

### 3. Process checklist

- [ ] **Shared resources**: If `jirachi/`, `zoroark/`, or `database/` changed, were all consumers verified?
- [ ] **Generated code regenerated**: If `queries_pregeneration/` changed, was `make generate` run and output committed?
- [ ] **PROGRESS.md**: If project state changed (feature completed, known issue fixed, roadmap shifted), is `PROGRESS.md` updated?
- [ ] **Implementation plan**: If harness items were completed or new items discovered, is `docs/harness/implementation-plan.md` updated?
- [ ] **PR ready**: Is the PR description drafted following `.github/PULL_REQUEST_TEMPLATE.md`?

### 4. Run verification

Run `make verify` from the repo root to confirm lint + typecheck + unit tests pass.

### 5. Report

Summarize findings:
- **Clean**: "Self-review passed, ready to commit."
- **Issues found**: List each issue with file and line. Fix before committing.

Do NOT commit. Report your findings and wait for instructions.
