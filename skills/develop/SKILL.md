---
name: kha:develop
description: Use when developing tasks in READY FOR DEVELOPMENT status. Creates a branch, implements using TDD with small commits, and moves to IN REVIEW. Processes ONE task per invocation.
---

# kha: Develop

> **ONE TASK PER INVOCATION.** Call `$KHA next` exactly once. All task data — description, comments, kha_blocks — is in the returned JSON. Never call `$KHA next` again. Never fetch tasks or comments separately.

Finds one `type:task` or `type:bug` in `READY FOR DEVELOPMENT`, implements it on a dedicated branch using TDD, and moves to `IN REVIEW`.

## Context

Read `AGENTS.md` once → note `list_id` and the pipeline order (the `→`-separated statuses in the Pipeline section).

Do NOT read the ClickUp Pipeline or Taxonomy docs — they are not needed.

## AWAITING INPUT Status

If `AWAITING INPUT` does not exist in the list, create it once via `mcp__clickup__clickup_update_list` (orderindex before BACKLOG, color `#e8a838`). Reuse — do not recreate.

## No Silent Assumptions

Never assume architecture or implementation scope. When ambiguous: state observation, present suggestion, wait for explicit agreement.

## Steps

**Step 1 — Fetch all READY FOR DEVELOPMENT tasks (run this entire block as one bash command):**

```bash
KHA="$HOME/.kha/kha"; [ -f .env.local ] && source .env.local
"$KHA" next "ready for development" --list <LIST_ID> --pipeline "<PIPELINE>"
```

Replace `<LIST_ID>` with the list ID from `AGENTS.md`. Replace `<PIPELINE>` with the pipeline from `AGENTS.md` (the `→`-separated statuses, lowercased, comma-separated).

**If this command exits with an error → report the exact error text and stop. Do NOT retry.**

**The response JSON has this exact shape:**
```json
{
  "tasks": [
    {
      "id": "86e22abc",
      "name": "Task title",
      "status": "ready for development",
      "task_type": "task",
      "description": "...",
      "url": "https://app.clickup.com/t/86e22abc",
      "assignees": [{ "id": 123, "email": "user@example.com" }],
      "comments": [...],
      "kha_blocks": {
        "scoping": { "acceptance_criteria": ["..."] },
        "design:context": {
          "architecture": "...",
          "scope": "...",
          "acceptance_criteria": ["..."],
          "file_hints": "..."
        }
      }
    }
  ],
  "current_user": { "id": 123, "email": "user@example.com" },
  "advanced_features": [],
  "message": "(only present when tasks array is empty)"
}
```

If `tasks` is empty → report `message` + any `advanced_features` and stop.

---

**From this point, work entirely from the JSON above. Do NOT call `$KHA next` again.**

- Iterate `tasks` by index in your reasoning. No CLI calls on decline.
- `tasks[i].kha_blocks["design:context"]` has architecture, scope, criteria, and file hints.
- `tasks[i].kha_blocks.scoping` has user-facing criteria as fallback.

---

**Step 2 — Task selection:**

```bash
KHA_MODE="${KHA_MODE:-interactive}"
```

**If `KHA_MODE=auto`:** Find the first `tasks[i]` where `task_type` is `task` or `bug`. For each skipped, log: "Skipped [epic|feature] **[name]** — run kha:[scoping|design] first." If none found → report "No actionable tasks in READY FOR DEVELOPMENT" and stop. Report: "Auto-selected: **[name]** ([task_type])". Assign and start timer:
```bash
"$KHA" update <task.id> --start-timer --assign
```

**If `KHA_MODE=interactive` (default):**

- Start at `tasks[0]`. Check `task_type`:
  - **`epic`** → say "Epic — run kha:scoping first." Skip to next.
  - **`feature`** → say "Feature — run kha:design first." Skip to next.
  - **`task` or `bug`** → present: "**[name]** (`[task_type]`) — [scope from `kha_blocks["design:context"].scope` if present]. Work on this?"
- **Declined** → move to `tasks[1]`, etc. No CLI call.
- **All exhausted** → report "No actionable tasks remaining" and stop.
- **Confirmed** → assign and start timer:
  ```bash
  "$KHA" update <task.id> --start-timer --assign
  ```

**Step 2b — Resume check:**

If `tasks[i].kha_blocks["develop:question"]` absent → fresh start, continue to Step 3.

If `tasks[i].kha_blocks["develop:question"]` present:
- Find the human reply: first comment in `tasks[i].comments` after the question comment where `user.id ≠ current_user.id`
- **No reply found** → task was moved back before the human answered:
  ```bash
  "$KHA" update <task.id> --status "awaiting input"
  ```
  Report: "Task re-parked — no reply found yet." Stop.
- **Reply found** → use it to resolve the pending decision. Continue from the interrupted step (Step 3 or Step 6 — whichever posted the question).

**Step 3 — Check context:**

Use `tasks[i].kha_blocks["design:context"]` for criteria and file hints. Fall back to `tasks[i].kha_blocks.scoping`. If neither exists → post question comment via `mcp__clickup__clickup_create_comment`:
```
[kha:develop:question]
resume_status: in development
decision: no design/scoping context found
context: this task has no [kha:design:context] or [kha:scoping] block — cannot verify acceptance criteria or file hints
question: Proceed with development without context, or send back to design first?
options:
- proceed: implement without design context
- back to design: move this task back to READY FOR DEVELOPMENT
@<assignee username>
```
Then:
```bash
"$KHA" update <task.id> --status "awaiting input" --stop-timer
```
Stop.

**Step 4 — Move to IN DEVELOPMENT:**
```bash
"$KHA" update <task.id> --status "in development"
```

**Step 5 — Create branch:**
```bash
git checkout develop && git pull origin develop
git checkout -b task/<task.id>-<kebab-title>
```

**Step 6 — TDD loop** — for each acceptance criterion:
- **Red** → write failing test, run it, confirm it fails for the right reason. Commit: `test(<task.id>): <what it tests>`
- **Green** → implement minimum code to pass. Follow `file_hints`. Run all tests. Commit: `feat(<task.id>): <what was implemented>` (or `fix(...)` for bugs)
- If a structural decision arises → post question comment via `mcp__clickup__clickup_create_comment`:
  ```
  [kha:develop:question]
  resume_status: in development
  decision: structural decision required
  context: <what was being implemented and what the decision point is>
  question: <the specific architectural or structural question>
  @<assignee username>
  ```
  Then:
  ```bash
  "$KHA" update <task.id> --status "awaiting input" --stop-timer
  ```
  Stop. (On resume with reply: resolve the decision and continue the TDD loop from where it was interrupted.)
- If a test cannot be made green → report the blocker. Leave in IN DEVELOPMENT. Stop.

**Step 7 — Refactor pass** (optional). Run all tests again. Commit: `refactor(<task.id>): <what was cleaned>`

**Step 8 — Finalize:**
```bash
git push origin task/<task.id>-<kebab-title>
"$KHA" update <task.id> \
  --status "in review" \
  --comment "[kha:develop]\nbranch: task/<task.id>-<kebab-title>\ncriteria implemented:\n- <criterion> → <test name>" \
  --stop-timer
```

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Branch | task/[id]-[kebab-title] |
| Tests | [N] passing |
| Status | → IN REVIEW |
