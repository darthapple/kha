---
name: kha:scoping
description: Use when scoping tasks in BACKLOG or SCOPING status. Performs business analysis, detects epics and features, writes acceptance criteria, and moves to IN DESIGN. Processes ONE task per invocation.
---

# kha: Scoping

> **ONE TASK PER INVOCATION.** Pick the first task only (top of column by orderindex).
> After completing it, STOP. Never loop to the next task.
> Batch processing is forbidden — the user must re-invoke the skill for each task.

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

## Steps

1. **Find the task to process:**
   - First: fetch tasks in `SCOPING` status (`mcp__clickup__clickup_filter_tasks`). These are already in progress — resume them.
     - If found: sort by `orderindex` ascending, select `tasks[0]`. Skip steps 2–3 (already in SCOPING). Assign current user (see **Assignment Routine**) and start time tracking (see **Time Tracking**). Go to step 4.
   - If none in SCOPING: fetch tasks in `BACKLOG` status.
     - If none there either → report "Nothing to scope — no tasks in BACKLOG or SCOPING." Stop.
     - Sort by `orderindex` ascending, select `tasks[0]`.

2. Present the task to the user: "Found: **[title]** (ID: `[id]`). Process this task?" Wait for confirmation.

3. Move task to `SCOPING` status (doing state). Assign current user (see **Assignment Routine**). Start time tracking (see **Time Tracking**).

4. Fetch full task details: `mcp__clickup__clickup_get_task` (include `description`) + `mcp__clickup__clickup_get_task_comments`

5. **Route by task type:**

   ### type:epic
   - Propose a breakdown: present a numbered list of candidate `type:feature` child tasks (title + one-line description each)
   - Ask: "I'd break this epic into these features — does this look right before I create them?" Wait for answer.
   - On agreement: create each child as a `type:feature` task using `mcp__clickup__clickup_create_task`:
     `parent_id` = epic task ID, `status` = `BACKLOG`, `list_id` from AGENTS.md, `task_type` = `Feature`
   - Add `[kha:scoping:context]` comment to each child task:
     ```
     [kha:scoping:context]
     parent epic: <epic title> (<epic id>)
     business goal: <what this epic is trying to achieve>
     context: <relevant background the child task needs to be scoped independently>
     ```
   - The epic itself does NOT get acceptance criteria — those belong to the child features
   - Add comment to epic:
     ```
     [kha:scoping]
     type: epic
     routed: epic
     child features: <id>, <id>, ...
     ```
   - Move epic to `IN DESIGN`. Stop time tracking (see **Time Tracking**).

   ### type:feature
   - **Classify intent** — business or technical?
     - Technical = refactor, devops, infra with no user-facing behavior change
     - If ambiguous → state uncertainty, present reasoning, ask before proceeding
     - If clearly technical → add comment `[kha:scoping] routed: non-business → IN DESIGN`, move to `IN DESIGN`, stop time tracking (see **Time Tracking**), stop
   - **Business analysis** (business-routed features):
     - Write **user-facing** acceptance criteria: each is user-visible, testable, unambiguous, starts with a verb
       (e.g., "User receives a reset email when requesting password reset")
     - Identify user roles affected
     - If UI interaction is non-trivial → ask: "I'd like to create a wireframe/low-level design doc before proceeding. Should I?" Wait for answer.
     - If agreed → create a ClickUp doc with wireframes and flow description, link in comment
   - Add comment to task:
     ```
     [kha:scoping]
     type: feature
     routed: business
     affected roles: <comma-separated list>
     acceptance criteria:
     - <user-facing criterion — starts with verb>
     - <user-facing criterion — starts with verb>
     doc: <url if created, else omit this line>
     ```
   - Move task to `IN DESIGN`. Stop time tracking (see **Time Tracking**).

   ### type:task or type:bug
   - **Classify intent** — business or technical? (same rules as feature above)
   - If clearly technical → add comment `[kha:scoping] routed: non-business → IN DESIGN`, move to `IN DESIGN`, stop time tracking (see **Time Tracking**), stop
   - **Business analysis** (business-routed tasks):
     - Write **implementation-scope** acceptance criteria: technical, testable, specific
       (e.g., "POST /api/reset-password returns 200 with valid token")
     - Identify what this task delivers
   - Add comment to task:
     ```
     [kha:scoping]
     type: task
     routed: business
     acceptance criteria:
     - <implementation criterion — starts with verb>
     - <implementation criterion — starts with verb>
     ```
   - Move task to `IN DESIGN`. Stop time tracking (see **Time Tracking**).

6. **STOP.** Task is complete. Do not process any remaining tasks in the queue.
   One invocation = one task. The user must re-invoke `kha:scoping` for the next task.

## Assignment Routine

When starting work on a task, ensure the current user is assigned:
1. Call `mcp__clickup__clickup_get_workspace_members` and find the member with email `fernando.adriano@kheperi.com.br` — note their user ID. (Look up once per session and reuse.)
2. Check the task's existing `assignees` from the fetched task details.
3. If current user is **not** in the list: call `mcp__clickup__clickup_update_task` with `assignees` = all existing assignee IDs + current user ID.
4. If already assigned: skip.

## Time Tracking

**Start:** Call `mcp__clickup__clickup_start_time_tracking` with `task_id`. ClickUp automatically stops any previously active entry.

**Stop:** Call `mcp__clickup__clickup_stop_time_tracking`.

## Output

Report for the single processed task:

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [epic / feature / task / bug] |
| Routed | [business / non-business / epic] |
| Status | → IN DESIGN |
