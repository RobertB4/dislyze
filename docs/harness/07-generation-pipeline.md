# Workstream #7 — Generation Pipeline

## Current Setup

The only code generation tool in the repository is **SQLC** — SQL-to-Go compiler. There is no OpenAPI generation, protobuf, TypeScript codegen, or any other generation tooling.

### How generation works today

Each Go module runs SQLC independently:
- `lugia-backend/` → `make sqlc` → reads `queries_pregeneration/*.sql` + `../database/migrations` → outputs `queries/*.go`
- `giratina-backend/` → `make sqlc` → same pattern
- `jirachi/` → `make sqlc` → same pattern

All three modules share the same database schema (`../database/migrations`) but generate against their own subset of queries.

There is no root-level command that regenerates everything. No CI step validates that generated code is fresh.

### Frontend "generation"

Zoroark (shared UI library) runs `svelte-package` to compile Svelte components into a distributable `dist/` directory. Both frontends reference zoroark via `file:../zoroark`. This is a build step, not code generation in the traditional sense, but CI must run `npm run package` in zoroark before building frontends.

## What's Already Agent-Friendly

### 1. SQLC workflow is completely mechanical

Write SQL → `make sqlc` → get typed Go. The config files are declarative, the output is deterministic, and the generated files have clear `DO NOT EDIT` headers. An agent can follow this workflow without any ambiguity.

### 2. All three configs share the same structure

Every `sqlc.yaml` uses `version: "2"`, the same `gen.go` options (`emit_interface`, `emit_json_tags`, `emit_empty_slices`, etc.), and the same input/output pattern. An agent reading one config understands all three.

### 3. Generated code is committed to the repository

The `queries/` directories are tracked in git. An agent can read the generated code directly without needing to run generation first. This also means `git diff` after regeneration shows exactly what changed.

## What's NOT Agent-Friendly

### 1. No root-level `make generate` command (High Impact)

An agent modifying a SQL query in `lugia-backend/queries_pregeneration/` must know to `cd lugia-backend && make sqlc`. If the schema change also affects jirachi or giratina queries, the agent must independently run `make sqlc` in each affected module. There's no single command that regenerates everything atomically.

Worse: schema changes (in `database/migrations/`) affect all three modules, but there's no signal telling the agent which modules need regeneration. The agent must know the dependency graph.

**Improvement strategy:** Add a root-level `make generate` target that runs `make sqlc` in all three Go modules. This becomes the single command an agent runs after any SQL or schema change. The target should also be extensible — when OpenAPI generation is added later (see [#5 OpenAPI contract layer](05-openapi-contract-layer.md)), it slots into the same command.

### 2. No CI validation that generated code is fresh (High Impact)

An agent (or human) can modify a `.sql` file in `queries_pregeneration/`, commit without running `make sqlc`, and CI will pass. The old generated `.go` files still compile — they just don't reflect the SQL changes. This creates silent drift that manifests as runtime errors.

**Improvement strategy:** Add a CI step per Go module that runs `make sqlc` and then checks `git diff --exit-code queries/`. If there's a diff, the generated code is stale and CI fails. This is a standard "generation freshness" gate used in most codegen pipelines.

### 3. SQLC version skew across modules (Medium Impact)

Lugia and giratina were last generated with **sqlc v1.29.0**. Jirachi was generated with **sqlc v1.28.0**. An agent running `sqlc generate` in jirachi with a newer SQLC binary may produce output differences unrelated to the actual query changes — just from the version bump.

**Improvement strategy:** Pin the SQLC version in one place (e.g., a tool version file, a CI install step, or the root Makefile) and ensure all modules use the same version. Running `make generate` at the root should use a single SQLC binary for all modules.

### 4. `queries_pregeneration/` contains Go code, not just SQL (Medium Impact)

Lugia's `sqlc.yaml` has a type override that imports `UserRole` from `lugia/queries_pregeneration`. This means `queries_pregeneration/` is not purely a directory of SQL files — it also contains Go type definitions used by SQLC during generation. An agent adding a new query might not realize this directory has Go code, or might not know that custom type overrides exist.

**Improvement strategy:** Document in CLAUDE.md: "lugia's `queries_pregeneration/` contains both SQL files and Go type definitions (`UserRole`). When SQLC generates code, it uses these types via overrides in `sqlc.yaml`. Check `sqlc.yaml` overrides before adding columns with custom types." Alternatively, consider moving the Go type definitions to a separate file with a clear name (e.g., `types.go` in the same directory with a header comment explaining its role).

### 5. No generation for the frontend tier (Medium Impact — future gap)

Frontend TypeScript types are hand-written (see [#5 OpenAPI contract layer](05-openapi-contract-layer.md)). When the OpenAPI contract layer is introduced, the generation pipeline must extend to include TypeScript client generation. The current pipeline has no concept of cross-language generation — it's Go-only.

**Improvement strategy:** Design `make generate` from the start with extensibility in mind. A natural structure:

```makefile
generate:
	$(MAKE) generate-sqlc
	$(MAKE) generate-openapi  # future: spec → Go server + TS client

generate-sqlc:
	cd jirachi && make sqlc
	cd lugia-backend && make sqlc
	cd giratina-backend && make sqlc

generate-openapi:
	@echo "Not yet implemented"
```

This gives agents a single entry point (`make generate`) and makes the generation order explicit.

### 6. Zoroark must be built before frontend CI but this isn't obvious (Low Impact)

Both frontends depend on zoroark via `file:../zoroark`, and CI builds zoroark first. But there's no explicit dependency in the root Makefile or any documentation that explains this ordering. An agent setting up a new frontend or modifying zoroark might not know that `npm run package` in zoroark must run before `npm install` in the frontends.

**Improvement strategy:** Document the build dependency in CLAUDE.md. When `make generate` is created, consider including `cd zoroark && npm run package` as a step (even though it's technically a build, not generation — from an agent's perspective, "things that must run before the code is ready" should be in one command).

## Summary

| Finding | Impact | Action |
|---|---|---|
| No root-level `make generate` | High | Add root Makefile target that runs all generation |
| No CI freshness check for generated code | High | Add `sqlc generate` + `git diff --exit-code` in CI |
| SQLC version skew (v1.28.0 vs v1.29.0) | Medium | Pin single SQLC version across all modules |
| `queries_pregeneration/` contains Go code | Medium | Document in CLAUDE.md, consider clearer file naming |
| No frontend TypeScript generation | Medium | Design `make generate` to be extensible for OpenAPI |
| Zoroark build ordering not documented | Low | Document dependency, include in `make generate` |

The generation pipeline is minimal but functional — SQLC does exactly what it should. The two highest-leverage improvements are **adding a root-level `make generate`** (so agents have one command to run) and **adding CI freshness checks** (so stale generation is caught mechanically, not by human review). Both are straightforward to implement and immediately reduce agent error surface.
