<review-principles>
## Code Review / Refactoring Guidelines

- Read all related code to understand the full scope of the changes.
- Refactoring must not change behavior. If tests pass before, they must pass after with the same assertions.
- Check for scope creep. Are there changes unrelated to the stated goal? Remove them.
- Verify no unnecessary files, dependencies, or abstractions were added.
- Think big picture. Is the approach the correct one? Are there alternative approaches that are better?
  </review-principles>
