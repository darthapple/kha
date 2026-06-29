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

1. Read `AGENTS.md` → find `list_id`
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → get exact status names in order
3. Read Taxonomy doc (`_Config` space, doc ID: `2kza2py5-537`) → type definitions

## Steps

**Step 1 — Fetch all TRIAGE tasks (run this entire block as one bash command):**

```bash
_OS=$(uname -s 2>/dev/null || echo "Windows")
case "$_OS" in
  Darwin) [ "$(uname -m)" = "arm64" ] && KHA="$HOME/.kha/kha-darwin-arm64" || KHA="$HOME/.kha/kha-darwin-amd64" ;;
  Linux)  KHA="$HOME/.kha/kha-linux-amd64" ;;
  *)      KHA="$APPDATA/kha/kha.exe" ;;
esac
[ -f .env.local ] && source .env.local
"$KHA" next triage --list <LIST_ID> --pipeline "<STATUS_1>,<STATUS_2>,..."
```

Replace `<LIST_ID>` with the value from `AGENTS.md`. Replace the pipeline with the exact ordered status names from the Pipeline doc (comma-separated, lowercased).

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

**Step 2 — Selection loop** (iterate `tasks` array by index, no CLI calls):

- Start at `tasks[0]`. Present: "Found: **[name]** (`[task_type]`). Triage this task?"
- **Declined** → move to `tasks[1]`, `tasks[2]`, etc. No CLI call needed.
- **All exhausted** → report "No tasks remaining in TRIAGE" and stop.
- **Confirmed** → start timer and assign:
  ```bash
  "$KHA" update <task.id> --start-timer --assign
  ```
  Proceed to Step 3.

**Step 3 — Classify** using `tasks[i].description`, `tasks[i].comments`, `tasks[i].kha_blocks`:
- If classification is ambiguous → ask one focused question. Wait for answer.
- If `Bug` and no reproduction steps in description or comments → ask user for them. Wait.

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
