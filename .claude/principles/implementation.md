<implementation-principles>
## Implementation Guidelines

- Read before writing. Understand the existing code before modifying it.
- Follow existing patterns in the codebase. Check how similar things are already done before inventing a new approach.
- When renaming or restructuring a type/field, update ALL consumers in the same step — don't leave type errors for later.
- Verify your changes through observable behavior, not implementation artifacts. "It compiles" is not verification — verify that the behavior is correct. Tools at your disposal: playwright-cli for browser verification, curl for API endpoints, test commands for logic.
- Write scalable code. Don't be lazy and use the easiest solution. Always use the correct solution.
- Never silence complexity. If something can fail or produce unexpected results, make that visible — don't hide it to keep moving. For example: don't swallow errors with empty catch blocks, don't return default values when an operation failed, don't use `any` to avoid solving a type problem.
</implementation-principles>
