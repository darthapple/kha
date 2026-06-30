# kha — Agent Documentation

## What This Project Is

**kha** is a Claude Code plugin that automates a full software development pipeline by connecting Claude (the AI agent) to ClickUp (project management). When installed in a project, it provides seven slash commands — one per pipeline stage — that Claude can invoke to fetch tasks, process them, and advance them through the workflow autonomously or interactively.

The project lives at `github.com/darthapple/kha`. It is published as a Claude Code plugin via `.claude-plugin/plugin.json`.

---

## Architecture

Two components work together:

### 1. Go CLI binary (`~/.kha/kha`)

A small Go program (`cmd/kha/main.go`) that bridges skills to the ClickUp API. Skills always call this binary — they never call the ClickUp API directly. The binary has three commands:

| Command | Purpose |
|---------|---------|
| `kha next <status> --list <id> [--pipeline s1,s2,...]` | Fetches all tasks in a given status, sorts them hierarchically (parent → subtasks by orderindex), auto-advances feature parents whose children have all moved forward, and parses `[kha:*]` comment blocks. Returns a single JSON payload. |
| `kha update <task-id> [flags]` | Updates a task: `--status`, `--comment`, `--file`, `--assign`, `--start-timer`, `--stop-timer`. Returns JSON with the actions taken. |
| `kha cancel <task-id>` | Stops the running timer for the task's workspace. |

The binary is built for multiple platforms via `make all` and installed to `~/.kha/kha` (the canonical path all skills use) via `make install`.

### 2. Claude Code Skills (7 skills)

SKILL.md files under `skills/` define the instructions Claude follows when running each pipeline stage. Each skill is a self-contained procedure: call `kha next`, pick a task, process it, call `kha update` to advance it.

---

## Pipeline

Default order (low → high):

```
TRIAGE → BACKLOG → SCOPING → IN DESIGN → READY FOR DEVELOPMENT
    → IN DEVELOPMENT → IN REVIEW → TESTING → SHIPPED
```

Two side-track statuses are created on demand by skills:
- **AWAITING INPUT** — parked waiting for a human reply (before BACKLOG, color `#e8a838`)
- **MANUAL TESTING** — tasks that need human verification (after TESTING, color `#f4c430`)

The pipeline order can be overridden per-project via `--pipeline` flag or `~/.kha/config.json`.

---

## Skills Reference

Each skill processes **ONE task per invocation**. The pattern is identical across all skills:

1. Call `kha next <status>` — get JSON
2. Select a task (auto or interactive)
3. Resume check — detect pending `[kha:*:question]` blocks and look for a human reply
4. Process the task
5. Call `kha update` to advance status and write a `[kha:*]` result comment

### kha:issue
Creates a new task in TRIAGE status. Reads `AGENTS.md` for the list ID. Asks user for title and description. Uses `mcp__clickup__clickup_create_task`.

### kha:triage
Classifies tasks by type (Bug / Feature / Epic / Task) using the native ClickUp task_type field. Moves to BACKLOG. Asks a `[kha:triage:question]` comment if classification is ambiguous or a Bug lacks reproduction steps.

### kha:scoping
Writes acceptance criteria. Routes by intent:
- **Epic** → proposes feature breakdown, creates child Feature tasks in BACKLOG after human approval
- **Feature (business)** → writes user-facing acceptance criteria → moves to IN DESIGN
- **Feature (technical)** → moves directly to IN DESIGN
- **Task / Bug** → writes implementation-scope criteria → moves to IN DESIGN

Asks a `[kha:scoping:question]` comment when intent is ambiguous.

### kha:design
Read-only codebase analysis. **Never writes or edits any file.** Proposes architecture, gets human approval, then:
- **Feature** → breaks into `type:task` children at READY FOR DEVELOPMENT, adds `[kha:design:context]` comment to each
- **Task / Bug** → adds `[kha:design:context]` comment directly on the task → moves to READY FOR DEVELOPMENT

Asks `[kha:design:question]` for architecture approval and child task list approval.

### kha:develop
TDD implementation loop on a dedicated branch. Skips `epic` and `feature` tasks. For `task` and `bug`:
1. Moves to IN DEVELOPMENT
2. Creates branch `task/<id>-<kebab-title>` from `develop`
3. Red → Green → Refactor per acceptance criterion (each step is a commit)
4. Pushes branch, moves to IN REVIEW

Commit conventions: `test(<id>):`, `feat(<id>):`, `fix(<id>):`, `refactor(<id>):`

### kha:review
Checks out the task branch (`kha_blocks.develop.branch`), diffs against `develop`, and reviews in three layers:
1. Acceptance criteria (from `kha_blocks.scoping` or `kha_blocks["design:context"]`) — every criterion ✅ or ❌ with file:line
2. Best practices — idiomatic patterns, naming, dead code
3. Security — OWASP Top 10 relevant to the stack

**Pass** → moves to TESTING. **Fail** → leaves in IN REVIEW, writes `[kha:review]` comment with findings.

### kha:qa
Writes automated tests (unit/integration for `type:task`, Playwright e2e for `type:feature`). One test per criterion. After human confirmation, commits and runs all tests. On full pass:
- Merges task branch into `develop` (`--no-ff`)
- Deletes the task branch locally and remotely
- Creates a `develop → main` PR via `gh pr create` (or appends URL if PR already open)
- Moves to SHIPPED

If some criteria need manual testing → moves to MANUAL TESTING.

---

## KHA Blocks

Structured comment blocks are the primary mechanism for passing context between pipeline stages. The `kha next` command parses them automatically — skills read `tasks[i].kha_blocks` and never parse comments manually.

Format in ClickUp comments:
```
[kha:block-name]
key: value
list-key:
- item 1
- item 2
```

Key blocks written by each skill:

| Block | Written by | Contains |
|-------|-----------|----------|
| `[kha:triage]` | kha:triage | type, reasoning |
| `[kha:scoping]` | kha:scoping | type, routed, acceptance criteria |
| `[kha:scoping:context]` | kha:scoping | parent epic context for child features |
| `[kha:design]` | kha:design | architecture summary, child task IDs |
| `[kha:design:context]` | kha:design | architecture, scope, criteria, file hints (on child tasks) |
| `[kha:develop]` | kha:develop | branch name, criteria implemented |
| `[kha:review]` | kha:review | result (approved / changes requested), criteria, security findings |
| `[kha:qa]` | kha:qa | result, automated test count, coverage map, PR URL |
| `[kha:*:question]` | any skill | resume_status, decision, context, question, options |
| `[kha:auto]` | kha binary | auto-advance comment on feature parents |

---

## Async Human Loop

When a skill needs human input it cannot resolve autonomously:
1. Posts a `[kha:SKILL:question]` comment (via `mcp__clickup__clickup_create_comment`) with `resume_status`, `decision`, `question`, and `options`
2. Calls `kha update <id> --status "awaiting input" --stop-timer`
3. Stops

On the next invocation of the same skill, the resume check detects the pending question block and looks for the first comment by a different user (human) posted after it. If found, uses the reply to resolve the decision and continues from the interrupted step. If not found, re-parks to AWAITING INPUT and stops.

---

## Feature Auto-Advancement (`pipeline.AdvanceFeature`)

When `kha next` encounters a `type:feature` task, it calls `AdvanceFeature`, which:
1. Fetches the feature with all subtasks
2. Finds the minimum pipeline rank among all children
3. If all children have moved past the feature's current status, advances the feature to match the minimum child rank
4. Writes a `[kha:auto]` comment explaining the advance

Features that were advanced are excluded from the `tasks` array (they moved status) and reported in `advanced_features`. Features that were not advanced are included in `tasks` for the skill to handle.

---

## KHA_MODE

| Value | Behavior |
|-------|----------|
| `interactive` (default) | Presents each task to the user and waits for confirmation before processing |
| `auto` | Automatically selects `tasks[0]` (or first actionable task), no user confirmation |

Set via environment variable `KHA_MODE=auto`. Used in Docker container deployments.

---

## Configuration

Priority order for `CLICKUP_API_KEY`:
1. `CLICKUP_API_KEY` environment variable
2. `.env.local` in the current working directory
3. `~/.kha/config.json` → `CLICKUP_API_KEY` field

Full `~/.kha/config.json` schema:
```json
{
  "CLICKUP_API_KEY": "pk_...",
  "user_email": "user@example.com",
  "pipeline": ["triage", "backlog", "scoping", "in design", "ready for development", "in development", "in review", "testing", "shipped"]
}
```

---

## AGENTS.md (Project-Side File)

Every project that uses kha skills must have an `AGENTS.md` file at its root. Each skill reads this file first to get:
- `list_id` — the ClickUp list ID for the project
- Pipeline order — the `→`-separated status sequence

Example:
```markdown
## ClickUp

list_id: 901714686477

## Pipeline

TRIAGE → BACKLOG → SCOPING → IN DESIGN → READY FOR DEVELOPMENT → IN DEVELOPMENT → IN REVIEW → TESTING → SHIPPED
```

---

## Docker Deployment (Autonomous Mode)

`docker-compose.yml` runs two services: a **NATS** server (JetStream) and a **manager** container. The manager orchestrates the entire pipeline — no skill containers run permanently.

### How it works

1. Manager polls ClickUp every `POLL_INTERVAL` seconds for each automated pipeline step.
2. For each step with pending tasks, it checks NATS KV for a slot lock (`slots.<skill>`).
3. If the slot is free it acquires the lock and spawns an ephemeral Docker container for that skill.
4. The skill container runs once (`docker-entrypoint-once.sh`), clones the project repo, processes one task, and exits.
5. Manager receives the exit code, releases the slot, and logs the result.
6. NATS KV slot TTL (`SLOT_TTL`, default 30 min) auto-releases the lock if the container crashes.

### Entrypoints

| Script | Used by |
|--------|---------|
| `docker-entrypoint.sh` | Legacy single-container polling mode (kept for local testing) |
| `docker-entrypoint-once.sh` | Manager-spawned skill containers — runs once and exits |

### Images

| Image | Built from |
|-------|-----------|
| `ghcr.io/darthapple/kha:latest` | `Dockerfile` — skill image (Claude + kha binary + plugin files) |
| Manager binary | `Dockerfile.manager` — Go binary + kha binary for `kha next` calls |

### Required `.env.agents` secrets

**API-key auth** (Anthropic API):
```
CLICKUP_API_KEY=pk_...
ANTHROPIC_API_KEY=sk-ant-...
GIT_TOKEN=ghp_...        # needed for develop and qa skills
```

**OAuth auth** (Claude Code subscription — token generated with `claude setup-token`):
```
CLICKUP_API_KEY=pk_...
CLAUDE_CODE_OAUTH_TOKEN=<token>
GIT_TOKEN=ghp_...
```
Generate the token once and paste it into `.env.agents`:
```bash
claude setup-token
# copy the printed token value into CLAUDE_CODE_OAUTH_TOKEN
```
Requires a Pro, Max, Team, or Enterprise Claude subscription. No API key needed. No credential directory mounting needed.

### Manager env vars (set in `docker-compose.yml`)

| Var | Purpose | Default |
|-----|---------|---------|
| `PROJECT_REPO_URL` | Git URL of the project to work on | (required) |
| `CLICKUP_LIST_ID` | ClickUp list ID | (required) |
| `SKILL_IMAGE` | Docker image to spawn for skills | `ghcr.io/darthapple/kha:latest` |
| `NATS_URL` | NATS server address | `nats://nats:4222` |
| `POLL_INTERVAL` | Seconds between ClickUp polls | `60` |
| `SLOT_TTL` | Seconds before a slot lock auto-expires | `1800` |
| `PIPELINE_STEPS` | JSON override for pipeline step→skill mapping | (default 9-step pipeline) |

### Docker socket

The manager mounts `/var/run/docker.sock` to spawn skill containers via the Docker Engine API. For tighter security, consider fronting the socket with a proxy that allows only `containers/create`, `containers/start`, `containers/wait`, `containers/logs`, and `containers/remove`.

---

## Internal Package Structure

```
cmd/
  kha/main.go            — CLI entry point; NextResult, UpdateResult types; flag parsing
  manager/main.go        — Manager entry point; connects NATS, inits slot store, runs scheduler
internal/
  clickup/
    client.go            — HTTP client; ListTasks, GetTask, GetComments, UpdateTask,
                           AddComment, GetCurrentUser, StartTimer, StopTimer,
                           AssignUser, UploadAttachment
    types.go             — Task, TaskWithOrder, Status, Member, Comment, Team,
                           rawType (handles string|int|null task_type),
                           OrderIndex (string-decimal float)
  config/
    config.go            — Config{APIKey, UserEmail, Pipeline}; load from env → .env.local → ~/.kha/config.json
  executor/
    executor.go          — Executor interface: Run(ctx, skill) → RunResult
    docker.go            — Docker Engine API implementation over Unix socket (no SDK);
                           demultiplexes container logs
  manager/
    config.go            — ManagerConfig; PipelineStep{Status, Skill}; loads from env
    scheduler.go         — Poll → slot check → dispatch → slot release loop
  pipeline/
    pipeline.go          — Order (status rank map), SortHierarchical, ParseKhaBlocks,
                           TaskTypeName, AdvanceFeature
  slots/
    slots.go             — NATS JetStream KV slot store; Acquire (atomic Create),
                           IsOccupied, Release (Purge); TTL = crash recovery
```

External dependency: `github.com/nats-io/nats.go` (JetStream KV). Everything else is standard library.

---

## Git Workflow

- **Base branch**: `develop` (all work branches off here)
- **Task branches**: `task/<task-id>-<kebab-title>`
- **Merge to main**: via `develop → main` PR created by `gh pr create` at ship time
- **Commit conventions** (conventional commits scoped to task ID):
  - `test(<id>): ...` — failing test (red phase)
  - `feat(<id>): ...` — implementation (green phase)
  - `fix(<id>): ...` — bug fix (green phase)
  - `refactor(<id>): ...` — cleanup (refactor phase)

---

## Build & Install

```bash
make all          # builds dist/kha-{darwin-arm64,darwin-amd64,linux-amd64,windows-amd64.exe}
make install      # builds for current platform + copies to ~/.kha/kha
make manager      # builds dist/manager-linux-amd64 (for Dockerfile.manager)
make clean        # removes dist/
```

The `Dockerfile.manager` multi-stage build handles `manager` and `kha` binaries automatically — no manual `make manager` needed for Docker.

---

## Plugin Registration

The plugin is a directory with `.claude-plugin/plugin.json`. Claude Code loads it when the directory is listed in a project's or global `settings.json` under `"plugins"`. The entrypoint script does this automatically at container startup for Docker deployments. For local use, add the path to `~/.claude/settings.json` or the project's `.claude/settings.json`.
