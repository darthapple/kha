---
name: kha:scoping
description: Use when scoping tasks in BACKLOG or SCOPING status. Performs business analysis, detects epics and features, writes acceptance criteria, and moves to IN DESIGN. Processes ONE task per invocation.
---

# kha: Scoping

> **ONE TASK PER INVOCATION.** Iterate the ordered list; present the first task to the user. If the user declines, present the next. Process only one task per invocation — declining is selection, not processing.

Processes one task from `SCOPING` (resuming in-progress) or `BACKLOG` (starting new). Classifies intent, detects epics/features, performs business analysis, writes acceptance criteria, and moves to `IN DESIGN`.

## Task Type Hierarchy

```
Epic     — large initiative, broken into type:feature children
Feature  — user-facing chunk of value, scoped here then broken into type:task by kha:design
Task     — single implementation unit (standalone or created by kha:design)
Bug      — treated as type:task (leaf, goes directly to development)
```

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc ID, taxonomy doc ID
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
3. Read Taxonomy doc (`_Config` space, doc ID: `2kza2py5-537`) → label rules

## No Silent Assumptions

Never assume intent, scope, or classification. When anything is ambiguous:
1. State what you observed and why it is uncertain
2. Present your suggestion with reasoning
3. Wait for explicit agreement before acting

The only actions allowed without confirmation: reading data, moving to the doing state (`SCOPING`), and adding informational comments.

## Platform Setup

Run once per session, cache `$KHA`:
```bash
_OS=$(uname -s 2>/dev/null || echo "Windows")
case "$_OS" in
  Darwin) [ "$(uname -m)" = "arm64" ] && KHA=~/.kha/kha-darwin-arm64 || KHA=~/.kha/kha-darwin-amd64 ;;
  Linux)  KHA=~/.kha/kha-linux-amd64 ;;
  *)      KHA="$APPDATA/kha/kha.exe" ;;
esac
```

## Steps

1. **Find the task to process — try SCOPING first, fall back to BACKLOG:**
   ```bash
   result=$($KHA next scoping --list <LIST_ID>)
   # if task is null:
   result=$($KHA next backlog --list <LIST_ID>)
   ```
   If both return null → report "Nothing to scope — no tasks in BACKLOG or SCOPING." Stop.
   Timer starts automatically on whichever task is returned.

2. **Selection loop:**
   - Present: "Found: **[task.name]** (ID: `[task.id]`). Process this task?"
   - **Confirmed** → move to doing state and assign user:
     ```bash
     $KHA update <task.id> --status scoping --assign
     ```
     Proceed to step 3.
   - **Declined** → cancel timer and get next:
     ```bash
     $KHA cancel <task.id>
     result=$($KHA next <same-status> --list <LIST_ID> --skip <all,seen,ids>)
     ```
     If null → try the other status or report "No tasks remaining." Loop.

3. All context is in the JSON from step 1:
   - `task.name`, `task.description` — task content
   - `comments` array — full comment thread
   - `kha_blocks` — parsed `[kha:*]` comment blocks

4. **Route by task type** (`task.task_type` from JSON):

   ### type:epic
   - Propose a breakdown: present a numbered list of candidate `type:feature` child tasks (title + one-line description each)
   - Ask: "I'd break this epic into these features — does this look right before I create them?" Wait for answer.
   - On agreement: create each child as a `type:feature` task using `mcp__clickup__clickup_create_task`:
     `parent_id` = epic task ID, `status` = `BACKLOG`, `list_id` from AGENTS.md, `task_type` = `Feature`
   - Add `[kha:scoping:context]` comment to each child task via `mcp__clickup__clickup_create_comment`:
     ```
     [kha:scoping:context]
     parent epic: <epic title> (<epic id>)
     business goal: <what this epic is trying to achieve>
     context: <relevant background the child task needs to be scoped independently>
     ```
   - The epic itself does NOT get acceptance criteria — those belong to the child features
   - Finalize epic:
     ```bash
     $KHA update <task.id> \
       --status "in design" \
       --comment "[kha:scoping]\ntype: epic\nrouted: epic\nchild features: <id>, <id>, ..." \
       --stop-timer
     ```

   ### type:feature
   - **Classify intent** — business or technical?
     - Technical = refactor, devops, infra with no user-facing behavior change
     - If ambiguous → state uncertainty, present reasoning, ask before proceeding
     - If clearly technical:
       ```bash
       $KHA update <task.id> \
         --status "in design" \
         --comment "[kha:scoping]\nrouted: non-business → IN DESIGN" \
         --stop-timer
       ```
       Stop.
   - **Business analysis** (business-routed features):
     - Write **user-facing** acceptance criteria: each is user-visible, testable, unambiguous, starts with a verb
     - Identify user roles affected
     - If UI interaction is non-trivial → ask: "I'd like to create a wireframe/low-level design doc before proceeding. Should I?" If agreed → create a ClickUp doc, link in comment
   - Finalize:
     ```bash
     $KHA update <task.id> \
       --status "in design" \
       --comment "[kha:scoping]\ntype: feature\nrouted: business\naffected roles: <roles>\nacceptance criteria:\n- <criterion>\n- <criterion>" \
       --stop-timer
     ```

   ### type:task or type:bug
   - **Classify intent** — business or technical? (same rules as feature above)
   - If clearly technical:
     ```bash
     $KHA update <task.id> \
       --status "in design" \
       --comment "[kha:scoping]\nrouted: non-business → IN DESIGN" \
       --stop-timer
     ```
     Stop.
   - **Business analysis** (business-routed tasks):
     - Write **implementation-scope** acceptance criteria: technical, testable, specific
   - Finalize:
     ```bash
     $KHA update <task.id> \
       --status "in design" \
       --comment "[kha:scoping]\ntype: task\nrouted: business\nacceptance criteria:\n- <criterion>\n- <criterion>" \
       --stop-timer
     ```

5. Task complete. One invocation = one task scoped.

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [epic / feature / task / bug] |
| Routed | [business / non-business / epic] |
| Status | → IN DESIGN |
