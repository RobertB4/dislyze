# Task: [Title]

## Goal

Describe the task from the end user's perspective:

- Who is the user? (e.g., tenant admin, end customer, internal admin)
- What problem does this solve for them?
- What does the user experience look like before and after this change?

## Background

Why is this needed? What context is important for understanding the task?

## Architecture

How will this be implemented? Describe the approach at a high level:

- What components/layers are involved?
- What is the data flow?
- What existing patterns are being extended?
- What new files/types/endpoints are needed?

## Security Considerations

What are the security implications of this task? Think through attack vectors, trust boundaries, and data exposure risks.

## Performance Considerations

What are the performance implications of this task? Think through how this behaves at scale, with large datasets, and under concurrent access.

## Steps

Order steps by the customer journey — build from the user's experience inward.
Each step must include verification. What "verify" means depends on what changed:

- **Backend (lugia-backend, giratina-backend):** Write integration tests covering the new behavior. Tests must run and pass. Use curl to verify endpoints.
- **Frontend (lugia-frontend):** Write e2e tests covering the new behavior. Tests must run and pass. Use `playwright-cli` to verify visually.
- **Frontend (giratina-frontend):** Use `playwright-cli` to verify visually (no e2e test infrastructure yet).
- **Shared library (jirachi):** Write unit tests for new logic. Tests must run and pass.
- **Shared components (zoroark):** Verify in both consuming frontends.

- [ ] Step 1: [description]
  - **Verify:** [what kind of tests to write, what to check in the browser]

## Status Updates

_Agents: add a brief entry here during each periodic review._

## Discoveries

_Updated during implementation. Capture anything learned that wasn't anticipated._
