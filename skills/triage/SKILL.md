---
name: kha:triage
description: Use when triaging tasks in TRIAGE status. Classifies each by type using the native Task Type field, asks clarifying questions only when needed, and moves items to BACKLOG. Processes ONE task per invocation.
---

# kha: Triage

> **ONE TASK PER INVOCATION.** Call `$KHA next` exactly once. All task data — description, comments, kha_blocks — is in the returned JSON. Never call `$KHA next` again. Never fetch tasks or comments separately.

Processes one task in `TRIAGE` status. Classifies it by type and moves to `BACKLOG`.

## Classification Rules

| Native Task Type | When to use |
|-----------------|-------------|
| `Bug` | Something broken that should work. Requires reproduction steps. |
| `Feature` | New user-facing functionality that doesn't exist yet. |
| `Epic` | Large initiative grouping multiple Features. |
| `Task` | Everything else — docs, refactors, research, standalone work. |

## Context

Read `AGENTS.md` once → note `list_id` and the pipeline order (the `→`-separated statuses in the Pipeline section).

Do NOT read the ClickUp Pipeline or Taxonomy docs — they are not needed.

## AWAITING INPUT Status

If `AWAITING INPUT` does not exist in the list, create it once via `mcp__clickup__clickup_update_list` (orderindex before BACKLOG, color `#e8a838`). Reuse — do not recreate.

## Steps

**Step 1 — Fetch all TRIAGE tasks (run this entire block as one bash command):**

```bash
KHA="$HOME/.kha/kha"; [ -f .env.local ] && source .env.local
"$KHA" next triage --list <LIST_ID> --pipeline "<PIPELINE>"
```

Replace `<LIST_ID>` with the list ID from `AGENTS.md`. Replace `<PIPELINE>` with the pipeline from `AGENTS.md` (the `→`-separated statuses, lowercased, comma-separated: e.g. `triage,backlog,scoping,in design,...`).

**If this command exits with an error → report the exact error text and stop. Do NOT retry with different arguments.**

**The response JSON has this exact shape:**
```json
{
  "tasks": [
    {
      "id": "86e22abc",
      "name": "Task title",
      "status": "triage",
      "task_type": "bug",
      "description": "Full description text",
      "url": "https://app.clickup.com/t/86e22abc",
      "assignees": [{ "id": 123, "email": "user@example.com" }],
      "comments": [
        { "id": "c1", "text": "Comment text", "date": "1234567890", "user": { "id": 123 } }
      ],
      "kha_blocks": {
        "triage": { "type": "Bug", "reasoning": "..." },
        "scoping": { "routed": "business" }
      }
    }
  ],
  "current_user": { "id": 123, "email": "user@example.com" },
  "advanced_features": [],
  "message": "(only present when tasks array is empty)"
}
```

If `tasks` is empty → report `message` and stop.

---

**From this point, work entirely from the JSON above. Do NOT call `$KHA next` again.**

- `tasks` is an ordered array. Work through it index by index in your reasoning.
- `tasks[i].comments` already contains every comment — never fetch comments separately.
- `tasks[i].kha_blocks` has pre-parsed `[kha:*]` blocks — never parse comments manually.
- `current_user.id` is your user ID (needed for `--assign`).

---

**Step 2 — Task selection:**

```bash
KHA_MODE="${KHA_MODE:-interactive}"
```

**If `KHA_MODE=auto`:** Take `tasks[0]`. If `tasks` is empty → report "No tasks in TRIAGE" and stop. Report: "Auto-selected: **[name]** ([task_type])". Start timer and assign:
```bash
"$KHA" update <task.id> --start-timer --assign
```
Proceed to Step 2b.

**If `KHA_MODE=interactive` (default):** (iterate `tasks` array by index, no CLI calls)

- Start at `tasks[0]`. Present: "Found: **[name]** (`[task_type]`). Triage this task?"
- **Declined** → move to `tasks[1]`, `tasks[2]`, etc. No CLI call needed.
- **All exhausted** → report "No tasks remaining in TRIAGE" and stop.
- **Confirmed** → start timer and assign:
  ```bash
  "$KHA" update <task.id> --start-timer --assign
  ```
  Proceed to Step 2b.

**Step 2b — Resume check:**

If `tasks[i].kha_blocks["triage:question"]` absent → fresh start, continue to Step 3.

If `tasks[i].kha_blocks["triage:question"]` present:
- Find the human reply: first comment in `tasks[i].comments` after the question comment where `user.id ≠ current_user.id`
- **No reply found** → task was moved back before the human answered:
  ```bash
  "$KHA" update <task.id> --status "awaiting input"
  ```
  Report: "Task re-parked — no reply found yet." Stop.
- **Reply found** → use it to resolve the pending decision. Continue to Step 4 (skip Step 3).

**Step 3 — Classify** using `tasks[i].description`, `tasks[i].comments`, `tasks[i].kha_blocks`:

If classification is clear → proceed to Step 4.

If type is ambiguous OR task is `Bug` with no reproduction steps in description or comments:
- Post question comment via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:triage:question]
  resume_status: triage
  decision: <type classification | reproduction steps>
  context: <what is missing or ambiguous>
  question: <specific question>
  options:
  - <option 1>
  - <option 2>
  @<assignee username>
  ```
- Then:
  ```bash
  "$KHA" update <task.id> --status "awaiting input" --stop-timer
  ```
  Stop.

**Step 4 — Write result:**
```bash
"$KHA" update <task.id> \
  --status backlog \
  --comment "[kha:triage]\ntype: <Type>\nreasoning: <one-line reasoning>" \
  --stop-timer
```

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [Bug / Feature / Epic / Task] |
| Status | → BACKLOG |
