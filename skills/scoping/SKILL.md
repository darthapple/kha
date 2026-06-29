---
name: kha:scoping
description: Use when scoping tasks in BACKLOG or SCOPING status. Performs business analysis, writes acceptance criteria, and moves to IN DESIGN. Processes ONE task per invocation.
---

# kha: Scoping

> **ONE TASK PER INVOCATION.** Call `$KHA next` at most twice (one per status). All task data — description, comments, kha_blocks — is in the returned JSON. Never call `$KHA next` again. Never fetch tasks or comments separately.

Processes one task from `SCOPING` (resuming) or `BACKLOG` (new). Writes acceptance criteria and moves to `IN DESIGN`.

## Task Type Hierarchy

```
Epic     — broken into type:feature children by this skill
Feature  — scoped here, broken into type:task children by kha:design
Task     — leaf node; gets implementation-scope criteria
Bug      — leaf node; treated like type:task
```

## Context

Read `AGENTS.md` once → note `list_id` and the pipeline order (the `→`-separated statuses in the Pipeline section).

Do NOT read the ClickUp Pipeline or Taxonomy docs — they are not needed.

## AWAITING INPUT Status

If `AWAITING INPUT` does not exist in the list, create it once via `mcp__clickup__clickup_update_list` (orderindex before BACKLOG, color `#e8a838`). Reuse — do not recreate.

## No Silent Assumptions

Never assume intent or scope. When ambiguous: state what you observed, present your suggestion with reasoning, wait for explicit agreement.

## Steps

**Step 1 — Fetch tasks (try SCOPING first, fall back to BACKLOG):**

Run the first bash block. If `tasks` is empty, run the second.

Replace `<LIST_ID>` with the list ID from `AGENTS.md`. Replace `<PIPELINE>` with the pipeline from `AGENTS.md` (the `→`-separated statuses, lowercased, comma-separated: e.g. `triage,backlog,scoping,...`). Use the same values in both blocks.

**If either command exits with an error → report the exact error text and stop. Do NOT retry.**

```bash
KHA="$HOME/.kha/kha"; [ -f .env.local ] && source .env.local
"$KHA" next scoping --list <LIST_ID> --pipeline "<PIPELINE>"
```

```bash
# Only if scoping returned empty tasks:
KHA="$HOME/.kha/kha"; [ -f .env.local ] && source .env.local
"$KHA" next backlog --list <LIST_ID> --pipeline "<PIPELINE>"
```

If both return empty `tasks` → report "Nothing to scope" and stop.

**The response JSON has this exact shape:**
```json
{
  "tasks": [
    {
      "id": "86e22abc",
      "name": "Task title",
      "status": "backlog",
      "task_type": "feature",
      "description": "Full description text",
      "url": "https://app.clickup.com/t/86e22abc",
      "assignees": [{ "id": 123, "email": "user@example.com" }],
      "comments": [
        { "id": "c1", "text": "Comment text", "date": "1234567890", "user": { "id": 123 } }
      ],
      "kha_blocks": {
        "triage": { "type": "Feature", "reasoning": "..." },
        "scoping:context": { "parent_epic": "...", "business_goal": "..." }
      }
    }
  ],
  "current_user": { "id": 123, "email": "user@example.com" },
  "advanced_features": [],
  "message": "(only present when tasks array is empty)"
}
```

---

**From this point, work entirely from the JSON above. Do NOT call `$KHA next` again.**

- Iterate `tasks` by index in your reasoning. No CLI calls on decline.
- `tasks[i].comments` and `tasks[i].kha_blocks` are fully loaded — never fetch separately.

---

**Step 2 — Selection loop:**

- Start at `tasks[0]`. Present: "Found: **[name]** (`[task_type]`). Process this task?"
- **Declined** → move to `tasks[1]`, etc. No CLI call.
- **All exhausted** → try the other status (if not yet tried), or stop.
- **Confirmed** → move to doing state, assign, start timer:
  ```bash
  "$KHA" update <task.id> --status scoping --start-timer --assign
  ```

**Step 2b — Resume check:**

If `tasks[i].kha_blocks["scoping:question"]` absent → fresh start, continue to Step 3.

If `tasks[i].kha_blocks["scoping:question"]` present:
- Find the human reply: first comment in `tasks[i].comments` after the question comment where `user.id ≠ current_user.id`
- **No reply found** → task was moved back before the human answered:
  ```bash
  "$KHA" update <task.id> --status "awaiting input"
  ```
  Report: "Task re-parked — no reply found yet." Stop.
- **Reply found** → use it to resolve the pending decision. Continue from the interrupted step in Step 4 (skip the part that triggered the question).

**Step 3 — All context is in `tasks[i]`:**
- `tasks[i].description`, `tasks[i].comments` — task content
- `tasks[i].kha_blocks["scoping:context"]` — parent epic context if applicable

**Step 4 — Route by `tasks[i].task_type`:**

### type:epic
- Propose breakdown: numbered list of `type:feature` children (title + one-line description)
- Post question comment via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:scoping:question]
  resume_status: scoping
  decision: epic breakdown approval
  context: proposed feature breakdown for this epic
  question: Does this breakdown look correct? Reply "approved" or describe changes.
  proposal:
  1. <Feature title> — <one-line description>
  2. <Feature title> — <one-line description>
  @<assignee username>
  ```
- Then:
  ```bash
  "$KHA" update <task.id> --status "awaiting input" --stop-timer
  ```
  Stop. (On resume with approval: proceed to create children below.)
- On approved reply: create each child via `mcp__clickup__clickup_create_task`:
  `parent_id` = epic ID, `status` = `BACKLOG`, `list_id` from AGENTS.md, `task_type` = `Feature`
- Add `[kha:scoping:context]` comment to each child via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:scoping:context]
  parent epic: <title> (<id>)
  business goal: <what this epic achieves>
  context: <relevant background>
  ```
- Finalize:
  ```bash
  "$KHA" update <task.id> \
    --status "in design" \
    --comment "[kha:scoping]\ntype: epic\nchild features: <id>, <id>, ..." \
    --stop-timer
  ```

### type:feature
- Classify intent — business (user-facing) or technical (no behavior change)?
- If clearly technical → move directly to IN DESIGN:
  ```bash
  "$KHA" update <task.id> --status "in design" --comment "[kha:scoping]\nrouted: non-business" --stop-timer
  ```
- If clearly business → write **user-facing** acceptance criteria (user-visible, testable, starts with verb):
  ```bash
  "$KHA" update <task.id> \
    --status "in design" \
    --comment "[kha:scoping]\ntype: feature\nrouted: business\naffected roles: <roles>\nacceptance criteria:\n- <criterion>" \
    --stop-timer
  ```
- If intent is ambiguous → post question comment via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:scoping:question]
  resume_status: scoping
  decision: intent classification
  context: <what is ambiguous about business vs technical intent>
  question: Is this user-facing (business) or purely technical (no behavior change)?
  options:
  - business: user-facing, needs acceptance criteria
  - technical: no behavior change, route directly to design
  @<assignee username>
  ```
  Then:
  ```bash
  "$KHA" update <task.id> --status "awaiting input" --stop-timer
  ```
  Stop.

### type:task or type:bug
- Same intent classification as feature.
- If clearly business → write **implementation-scope** criteria (technical, testable, specific):
  ```bash
  "$KHA" update <task.id> \
    --status "in design" \
    --comment "[kha:scoping]\ntype: task\nrouted: business\nacceptance criteria:\n- <criterion>" \
    --stop-timer
  ```
- If intent is ambiguous → same async question pattern as feature above (resume_status: scoping).

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [epic / feature / task / bug] |
| Status | → IN DESIGN |
