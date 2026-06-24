---
name: kha:backlog-triage
description: Use when triaging tasks in TRIAGE status. Classifies each by type using the native Task Type field, asks clarifying questions only when needed, and moves items to BACKLOG. Processes ONE task per invocation.
---

# kha: Backlog Triage

> **ONE TASK PER INVOCATION.** Pick the first task only (top of column by orderindex).
> After completing it, STOP. Never continue to the next task.
> Batch processing is forbidden — the user must re-invoke the skill for each task.

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

**Important:** Classification is set using the native ClickUp Task Type field (`task_type` in `mcp__clickup__clickup_update_task`), not tags or labels.

## Steps

1. Fetch all tasks in `TRIAGE` from the current list using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in TRIAGE" and stop.
   Sort the returned tasks by their `orderindex` field ascending. Select `tasks[0]` only.

3. Present the task to the user: "Found: **[title]** (ID: `[id]`). Triage this task?" Wait for confirmation.

4. Fetch full task details and comment thread using `mcp__clickup__clickup_get_task` (include `description`) and `mcp__clickup__clickup_get_task_comments` — read both before classifying.

5. Classify type using the rules above (consider title, description, and any comments).
   - If classification is ambiguous → ask user one focused question before continuing. Wait for answer.
   - If `Bug` and no reproduction steps in description or comments → ask user for them before continuing. Wait for answer.

6. Set the native task type using `mcp__clickup__clickup_update_task` with `task_type` = `Bug`, `Feature`, `Epic`, or `Task`.

7. Add comment: `[kha:triage] type: <type> — <one-line reasoning>`

8. Move task to `BACKLOG` status using `mcp__clickup__clickup_update_task`.

9. **STOP.** Do not process any remaining tasks in the queue.
   One invocation = one task. The user must re-invoke `kha:backlog-triage` for the next task.

## Clarifying Questions

- Ask only when classification is genuinely unclear from title, description, and comments
- One question per task, not a list of questions
- Wait for answer before proceeding

## Output

Report for the single processed task:

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [Bug / Feature / Epic / Task] |
| Status | → BACKLOG |
