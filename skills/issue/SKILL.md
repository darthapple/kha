---
name: kha:issue
description: Use when creating a new task in the current ClickUp project. Collects title and description from the user, creates the task in TRIAGE status.
---

# kha: Issue

Creates a new task in the current project's ClickUp list with TRIAGE status.

## Context

Read `AGENTS.md` in the current project to find the list ID. If no `AGENTS.md` exists, ask the user which ClickUp list to use before proceeding.

## Required Information

Gather before creating:
- **Title** (required) — clear, action-oriented. Ask if not provided.
- **Description** (optional) — context, acceptance criteria, or reproduction steps for bugs.

Do not assume the title. If missing, ask.

## Steps

1. Read `AGENTS.md` to get the current list ID
2. Ask user for title and description if not already provided
3. Create task with `mcp__clickup__clickup_create_task`:
   - `list_id`: from AGENTS.md
   - `name`: task title
   - `description`: task description (if provided)
   - `status`: `TRIAGE`
4. Report: task name, ID, and ClickUp URL

## Output

One line: `Created "[task name]" → [URL]`

Nothing else unless the user asks.
