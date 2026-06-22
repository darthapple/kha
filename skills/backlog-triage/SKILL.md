---
name: kha:backlog-triage
description: Use when triaging tasks in TRIAGE status. Classifies each by type, asks clarifying questions only when needed, and moves items to BACKLOG.
---

# kha: Backlog Triage

Processes all tasks in TRIAGE status for the current project. Classifies each by type and moves to BACKLOG.

## Context

Read `AGENTS.md` in the current project to find the list ID.
Read the Taxonomy document (`_Config` space, doc ID: `2kza2py5-537`) for full type definitions.

## Classification Rules

| Type | When to use |
|------|-------------|
| `bug` | Something broken that should work. Requires reproduction steps. |
| `feature` | New functionality that doesn't exist yet. |
| `epic` | Large initiative grouping multiple tasks or features. |
| `task` | Everything else — docs, refactors, research. |

## Steps

1. Fetch all tasks in `TRIAGE` from the current list using `mcp__clickup__clickup_filter_tasks`
2. If no tasks in TRIAGE, report "No items in TRIAGE" and stop
3. For each task:
   - a. Read title and description
   - b. Classify type using rules above
   - c. If classification is ambiguous → ask user one focused question before continuing
   - d. If `bug` and no reproduction steps in description → ask user before continuing
   - e. Add comment: `[kha:triage] type: <type> — <one-line reasoning>`
   - f. Move task to `BACKLOG` status using `mcp__clickup__clickup_update_task`
4. Report summary

## Clarifying Questions

- Ask only when classification is genuinely unclear from title + description
- One question per task, not a list of questions
- Wait for answer before moving to the next task

## Output

Summary table after all items are processed:

| Task | Type | Status |
|------|------|--------|
| Fix login redirect | bug | → BACKLOG |
| Add CSV export | feature | → BACKLOG |
