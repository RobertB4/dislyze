<debugging-principles>
## Debugging Guidelines

- Reproduce the issue first. Before changing anything, confirm you can trigger the bug and describe the expected vs actual behavior.
- Question your assumptions. The bug is rarely where you first think it is. Verify each assumption with evidence before acting on it.
- Find the root cause before writing a fix. Don't patch symptoms — understand WHY it's broken.
- Trace the data flow through the system to find where it diverges from expected behavior.
- After fixing, verify the fix resolves the issue AND doesn't break anything else. Verify through observable behavior, not just "it compiles."
</debugging-principles>
