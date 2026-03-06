<planning-principles>
## Planning Guidelines

- Start from the end user's perspective. What value does this feature produce for the user? Plan from that outcome backward — not from the technical implementation forward.
- Optimize for correct, extendable architecture. Never choose a quick hack over a clean design. If the clean approach takes more steps, that's fine.
- Trace the full data flow before designing. Identify every layer the data passes through and plan changes at each layer.
- Every step must be verified through observable behavior, not implementation artifacts. "The code compiles" or "the linter passes" is not verification — verify that the behavior is correct. Tools at your disposal: playwright-cli for browser verification, curl for API endpoints, test commands for logic.
- Identify ALL consumers. Search for every usage of the thing you're changing. List them. Plan how each one will be updated.
- Prefer fewer, larger steps that produce a working state over many small steps that leave things broken in between.
- Call out what does NOT change. Explicitly stating unchanged boundaries prevents scope creep.
</planning-principles>
