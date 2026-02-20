Stop what you're doing and zoom out. You may be stuck in a local maximum — solving the wrong problem, over-engineering, or missing a simpler approach. Work through these steps honestly.

## 1. State the problem

In one sentence: what are you trying to solve right now? Not what code you're writing — what *problem* you're solving.

## 2. Why does this problem exist?

Trace it upstream. Is what you're dealing with a root cause, or a symptom of something deeper? If you're adding a workaround, what would a real fix look like?

## 3. Map the blast radius

What parts of the system does your current approach touch? What are the second-order effects — what else changes, breaks, or gets more complex because of your approach?

## 4. Consider alternatives

Take a step back from your current implementation. Are there simpler approaches at a higher level? Could a different design eliminate the problem entirely instead of working around it? Think about:
- Could this be solved with a configuration change instead of code?
- Could an existing abstraction be extended instead of building something new?
- Is there a pattern elsewhere in the codebase that already solves this class of problem?

## 5. Question the premise

Should this even be done this way? Is the task itself well-defined, or are you filling in gaps with assumptions? If you're unsure about the direction, that's a signal to escalate — not to guess.

## 6. Decide

Based on your analysis, choose one:
- **Continue**: Your current approach is sound — explain why.
- **Pivot**: You've identified a better approach — describe it.
- **Escalate**: The task needs clarification or a decision you can't make — explain what you need from the human.

Present your analysis and wait for the human to steer.
