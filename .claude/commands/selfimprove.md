You are an expert in prompt engineering, specializing in optimizing AI code assistant instructions. Your task is to analyze this session and propose improvements to agent-facing files.

## Critical Rule: Abstract Over Concrete

Improvements MUST be abstract, transferable principles — NOT concrete facts about a specific task.

| Bad (concrete) | Good (abstract) |
|---|---|
| "The IP whitelist feature uses CIDR notation" | "When working with network inputs, validate both IPv4 and IPv6 formats" |
| "SSO endpoints need keycloak running" | "Features with external service dependencies should document their dev setup requirements" |
| "The auth middleware must come before feature checks" | "Middleware ordering is load-bearing — document the required order when adding new middleware" |

Ask yourself: "Would this help an agent working on a completely different task?" If the answer is no, make it more abstract.

## Process

### 1. Analysis Phase

Review the chat history in your context window. Then examine the current instructions:

<claude_instructions>
@CLAUDE.md
@database/CLAUDE.md
@lugia-backend/CLAUDE.md
@giratina-backend/CLAUDE.md
@lugia-frontend/CLAUDE.md
@giratina-frontend/CLAUDE.md
@jirachi/CLAUDE.md
@zoroark/CLAUDE.md
</claude_instructions>

Identify moments where:
- The agent made a wrong assumption that better instructions would have prevented
- The agent needed human correction that a principle could have avoided
- The agent didn't know something that should be documented as a general rule
- An existing instruction was unclear or misleading

### 2. Interaction Phase

Present your findings. For each suggestion:
a) What happened in this session (the concrete trigger)
b) The abstract principle that would prevent this class of problem
c) Which file it belongs in and why (usually a CLAUDE.md, but could be a command, Makefile, PR template, etc.)

Wait for human feedback before implementing.

### 3. Implementation Phase

For each approved change, edit the appropriate file. Place the improvement in the most relevant existing section, or create a new section if none fits.

Remember: the goal is to teach mental models that help agents derive correct behavior in novel situations — not to document facts about specific features.
