---
name: kha:triage
description: Use when triaging tasks in TRIAGE status. Classifies each by type using the native Task Type field, asks clarifying questions only when needed, and moves items to BACKLOG. Processes ONE task per invocation.
---

# kha: Triage

> **ONE TASK PER INVOCATION.** Iterate the ordered list; present the first task to the user. If the user declines, present the next. Process only one task per invocation — declining is selection, not processing.

Processes one task in `TRIAGE` status. Classifies it by type (sets native ClickUp Task Type field) and moves to `BACKLOG`.

## Context

1. Read `AGENTS.md` in the current project to find the list ID
2. Read the Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) for current status names
3. Read the Taxonomy doc (`_Config` space, doc ID: `2kza2py5-537`) for full type definitions

## Classification Rules

| Native Task Type | When to use |
|-----------------|-------------|
| `Bug` | Something broken that should work. Requires reproduction steps. |
| `Feature` | New user-facing functionality that doesn't exist yet. Gets broken into Tasks by kha:design. |
| `Epic` | Large initiative grouping multiple Features. Gets broken into Features by kha:scoping. |
| `Task` | Everything else — docs, refactors, research, standalone implementation work. |

## Platform Setup

Run once per session, cache `$KHA` and `$PIPELINE`:
```bash
_OS=$(uname -s 2>/dev/null || echo "Windows")
case "$_OS" in
  Darwin) [ "$(uname -m)" = "arm64" ] && KHA=~/.kha/kha-darwin-arm64 || KHA=~/.kha/kha-darwin-amd64 ;;
  Linux)  KHA=~/.kha/kha-linux-amd64 ;;
  *)      KHA="$APPDATA/kha/kha.exe" ;;
esac
```

After reading the Pipeline doc (Context step 2), extract the ordered status names and set `$PIPELINE` — comma-separated, lowercased, exact names from the doc in pipeline order:
```bash
PIPELINE="triage,backlog,scoping,in design,ready for development,in development,in review,testing,shipped"
```

## Steps

1. Fetch the first TRIAGE task (timer starts automatically):
   ```bash
   result=$($KHA next triage --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   Parse `result` as JSON. If `task` is null → report `message` and stop.

2. **Selection loop:**
   - Present: "Found: **[task.name]** (ID: `[task.id]`). Triage this task?"
   - **Confirmed** → proceed to step 3.
   - **Declined** → cancel timer and get next:
     ```bash
     $KHA cancel <task.id>
     result=$($KHA next triage --list <LIST_ID> --pipeline "$PIPELINE" --skip <all,seen,ids>)
     ```
     If task null → report "No tasks remaining in TRIAGE" and stop. Otherwise loop.

3. Classify type using the rules above. All context is in the JSON:
   - `task.name`, `task.description` — task content
   - `comments` array — full comment thread
   - If classification is ambiguous → ask one focused question. Wait for answer.
   - If `Bug` and no reproduction steps in description or comments → ask user for them. Wait for answer.

4. Write result (sets type, assigns current user, adds comment, moves to BACKLOG, stops timer):
   ```bash
   $KHA update <task.id> \
     --status backlog \
     --comment "[kha:triage]\ntype: <Type>\nreasoning: <one-line reasoning>" \
     --assign \
     --stop-timer
   ```

## Clarifying Questions

- Ask only when classification is genuinely unclear from title, description, and comments
- One question per task, not a list of questions
- Wait for answer before proceeding

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [Bug / Feature / Epic / Task] |
| Status | → BACKLOG |
