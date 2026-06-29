---
name: kha:design
description: Use when designing tasks in IN DESIGN status. Analyzes codebase, defines architecture, breaks features into type:task children, and moves to READY FOR DEVELOPMENT. Processes ONE task per invocation.
---

# kha: Design

> **ONE TASK PER INVOCATION.** Call `$KHA next` exactly once. All task data — description, comments, kha_blocks — is in the returned JSON. Never call `$KHA next` again. Never fetch tasks or comments separately.

Processes one task in `IN DESIGN` status. Analyzes the codebase, defines architecture, **documents the plan in ClickUp**, and moves to `READY FOR DEVELOPMENT`.

> **YOU ARE NOT A CODING AGENT.**
> This skill reads code to understand patterns — it never writes, edits, or creates any file.
> The only output is ClickUp comments and status updates.
> If you feel the urge to write code or edit a file: stop, capture the insight as text in the ClickUp comment instead, and continue.

## Context

Read `AGENTS.md` once → note `list_id` and the pipeline order (the `→`-separated statuses in the Pipeline section).

Do NOT read the ClickUp Pipeline or Taxonomy docs — they are not needed.

## No Silent Assumptions

Never assume architecture or scope. When ambiguous: state observation, present suggestion with reasoning, wait for explicit agreement.

## Steps

**Step 1 — Fetch all IN DESIGN tasks (run this entire block as one bash command):**

```bash
KHA="$HOME/.kha/kha"; [ -f .env.local ] && source .env.local
"$KHA" next "in design" --list <LIST_ID> --pipeline "<PIPELINE>"
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
      "status": "in design",
      "task_type": "bug",
      "description": "Full description text",
      "url": "https://app.clickup.com/t/86e22abc",
      "assignees": [{ "id": 123, "email": "user@example.com" }],
      "comments": [
        { "id": "c1", "text": "Comment text", "date": "1234567890", "user": { "id": 123 } }
      ],
      "kha_blocks": {
        "triage": { "type": "Bug", "reasoning": "..." },
        "scoping": { "routed": "business", "acceptance_criteria": ["..."] }
      }
    }
  ],
  "current_user": { "id": 123, "email": "user@example.com" },
  "advanced_features": [
    { "id": "xyz", "name": "Feature", "old_status": "in review", "new_status": "testing" }
  ],
  "message": "(only present when tasks array is empty)"
}
```

If `tasks` is empty → report `message` + any `advanced_features` and stop.
Report any `advanced_features` before continuing.

---

**From this point, work entirely from the JSON above. Do NOT call `$KHA next` again.**

- Iterate `tasks` by index in your reasoning. No CLI calls on decline.
- `tasks[i].comments` and `tasks[i].kha_blocks` are fully loaded — never fetch separately.
- `tasks[i].kha_blocks.scoping` has acceptance criteria written by kha:scoping.
- `tasks[i].kha_blocks["scoping:context"]` has parent epic context if applicable.

---

**Step 2 — Selection loop:**

- Start at `tasks[0]`. Check `task_type`:
  - **`epic`** → say "This is an epic — run kha:scoping first." Skip to `tasks[1]`.
  - **`feature`, `task`, `bug`** → present: "Found: **[name]** (`[task_type]`). Design this task?"
- **Declined** → move to `tasks[1]`, etc. No CLI call.
- **All exhausted** → report "No tasks to design" and stop.
- **Confirmed** → assign and start timer:
  ```bash
  "$KHA" update <task.id> --start-timer --assign
  ```

**Step 3 — Check for scoping context:**
- `tasks[i].kha_blocks.scoping` absent → confirm: "No scoping comment found. Proceed with technical design only, or send back to scoping?" Wait.

**Step 4 — Read the codebase for context only** — read relevant files, trace existing patterns, identify which files and functions are involved. **Do not edit, create, or write any file. Never implement anything.**

**Step 5 — Architecture proposal** — present the proposed approach in plain text (files to change, patterns to follow, edge cases). Always wait for explicit agreement before continuing. **Do not implement the proposal — document it in ClickUp in the next step.**

**Step 6 — Route by `task_type`:**

### type:feature
- Propose a numbered list of independent `type:task` children in plain text. Ask for confirmation. Wait. **Do not implement any of them — the list is documentation only.**
- On agreement: create each via `mcp__clickup__clickup_create_task`:
  `parent_id` = current task ID, `status` = `READY FOR DEVELOPMENT`, `list_id` from AGENTS.md, `task_type` = `Task`
- Add `[kha:design:context]` comment to each child via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:design:context]
  parent feature: <title> (<id>)
  architecture: <context for this task>
  scope: <exactly what this task covers>
  acceptance criteria:
  - <implementation criterion — starts with verb>
  file hints: <relevant files>
  ```
- Finalize:
  ```bash
  "$KHA" update <task.id> \
    --status "ready for development" \
    --comment "[kha:design]\narchitecture: <2-3 sentence summary>\nchild tasks: <id>, <id>, ..." \
    --stop-timer
  ```

### type:task or type:bug
- Define implementation approach in text only: which files to change, what the fix looks like, edge cases. **Write this plan as a ClickUp comment — never touch the actual files.**
- Add `[kha:design:context]` comment directly on this task via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:design:context]
  architecture: <approach summary>
  scope: <what this covers>
  acceptance criteria:
  - <implementation criterion — starts with verb>
  file hints: <relevant files>
  ```
- Finalize:
  ```bash
  "$KHA" update <task.id> \
    --status "ready for development" \
    --comment "[kha:design]\narchitecture: <2-3 sentence summary>" \
    --stop-timer
  ```

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [feature / task / bug] |
| Child Tasks | [N created / N/A] |
| Status | → READY FOR DEVELOPMENT |
