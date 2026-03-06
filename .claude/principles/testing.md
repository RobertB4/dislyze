<testing-principles>
## Testing Guidelines

- No mocks. Ever. Tests should be one of three kinds:
  1. Unit tests — pure functions with deterministic input/output. No side effects, no dependencies.
  2. Integration tests — test components working together with real dependencies (database, services).
  3. E2E tests — test the full system end-to-end using playwright and real services.
- Test edge cases and error paths, not just the happy path.
- Test behavior, not implementation details. Tests should not break when internal code is refactored.
  </testing-principles>
