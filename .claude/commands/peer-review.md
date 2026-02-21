Review a pull request as an independent reviewer. You have NOT seen this code before — question assumptions, don't just check a list.

If no PR number was provided in `$ARGUMENTS`, respond with: "Please provide a PR number: `/peer-review <pr-number>`" and stop.

## Steps

### 1. Gather context

Run these commands to understand the PR:

```bash
gh pr view $ARGUMENTS --json title,body,headRefName,baseRefName,files
gh pr diff $ARGUMENTS
```

Read the PR description carefully. If the description is empty or doesn't explain the change, flag that immediately.

### 2. Understand the change

Before reviewing line-by-line:
- What is this PR trying to accomplish?
- Does the approach make sense?
- Are there simpler alternatives the author may have missed?

### 3. Explore related code and documentation

Before judging the diff, build context:
- Read the CLAUDE.md files for each affected module
- Read the existing code surrounding the changed lines — understand what the code looked like before and why
- Check how similar patterns are implemented elsewhere in the codebase
- Read any documentation referenced in the PR description

This step is critical — reviewing a diff without understanding the codebase leads to shallow feedback.

### 4. Review the diff

For every changed file, evaluate:

- [ ] **Purpose**: Does this file change make sense for the stated goal?
- [ ] **Correctness**: Will this code actually work? Look for off-by-one errors, nil/undefined access, missing error handling, race conditions.
- [ ] **Conventions**: Does it follow the patterns in the relevant CLAUDE.md? Read the CLAUDE.md for each affected module.
- [ ] **Generated code**: Are `queries/` changes made via SQLC, not hand-edited?
- [ ] **Security**: No secrets, no injection (SQL, XSS, command), no SSRF, no auth bypass?
- [ ] **Blast radius**: If shared resources changed (jirachi, zoroark, database), are all consumers considered?
- [ ] **Tests**: If behavior changed, are tests added or updated?
- [ ] **Scope creep**: Are there unrelated changes bundled in?
- [ ] **Anything else**: If something feels off but isn't covered above, flag it. This checklist is not exhaustive — use your judgement.

### 5. Verify

If the branch is checked out locally, run `make verify`. If not, check if CI has passed on the PR:

```bash
gh pr checks $ARGUMENTS
```

### 6. Submit review

Based on your findings, submit the review directly:

- **Approve**: `gh pr review $ARGUMENTS --approve --body "..."`
- **Request changes**: `gh pr review $ARGUMENTS --request-changes --body "..."`
- **Comment only**: `gh pr review $ARGUMENTS --comment --body "..."`

The review body should summarize findings clearly so the PR author (human or agent) can act on them.
