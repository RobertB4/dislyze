Reflect on this session and record a score in the harness metrics.

## Process

### 1. Read the rubric

Read `docs/harness/metrics/sessions.js` to understand the scoring dimensions, difficulty scale, and existing session entries.

### 2. Reconstruct the session

Review the chat history in your context window. Identify:
- What was the task?
- What went well (autonomous, correct, clean)?
- What needed human correction? Be honest — every correction is a data point.
- How many CI/verification fix rounds were needed?
- Did the scope stay clean or drift?

### 3. Score honestly

For each dimension, assign a score with a brief justification. Resist the temptation to be generous — the value of this data comes from accuracy, not from high numbers. If the human corrected you, that's a 3, not a 5.

Use this mental model:
- **5**: No issues at all in this dimension
- **4**: Minor issues that didn't require human intervention
- **3**: Human had to correct something once
- **2**: Human had to correct something multiple times
- **1**: Significant problems throughout

### 4. Present for review

Show the scores and rationale in a table. Wait for human feedback before writing.

### 5. Write the entry

After human approval, append a new entry to the `sessions` array in `docs/harness/metrics/sessions.js`. Follow the exact format of existing entries.

The `notes` field should capture:
- What was done (brief)
- Specific human corrections (these are the most valuable data points)
- Any new principles or patterns that emerged

Do NOT inflate scores. Do NOT omit corrections from the notes. The goal is honest measurement so we can track improvement over time.
