---
name: kha:develop
description: Use when developing tasks in READY FOR DEVELOPMENT status. Picks the top type:task by column order, creates a branch, implements using TDD with small commits, and moves to IN REVIEW. Processes ONE task per invocation.
---

# kha: Develop

> **ONE TASK PER INVOCATION.** Pick the first task only (top of column by orderindex).
> After completing it, STOP. This skill already processes one task by design — maintain that discipline.

Picks the top `type:task` (or `type:bug`) in `READY FOR DEVELOPMENT`, implements it using TDD with small commits on a dedicated branch, and moves to `IN REVIEW`.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc ID, taxonomy doc ID
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm status names (`READY FOR DEVELOPMENT`, `IN DEVELOPMENT`, `IN REVIEW`)

## No Silent Assumptions

Never assume architecture, file structure, or implementation scope. When anything is ambiguous:
1. State what you observed and why it is uncertain
2. Present your suggestion with reasoning
3. Wait for explicit agreement before acting

The only actions allowed without confirmation: reading data, creating the branch, moving to `IN DEVELOPMENT`, and adding informational comments.

## Steps

1. Fetch tasks in `READY FOR DEVELOPMENT` in column order using curl (MCP strips `orderindex`). Get list ID from `AGENTS.md`, API key from `.env.local`:
   ```bash
   source .env.local && curl -s "https://api.clickup.com/api/v2/list/<LIST_ID>/task?statuses[]=ready%20for%20development&subtasks=true" -H "Authorization: $CLICKUP_API_KEY"
   ```
   Sort the returned `tasks` array by `orderindex` ascending — this reflects column order (top to bottom). Never reorder by age, priority, or any other field. Select `tasks[0]`.
2. If response contains no tasks → report "No items in READY FOR DEVELOPMENT" and stop.
3. Present the top task: title, ID, and one-line summary from its `[kha:design:context]` comment (if present). Ask: "Work on this task?" Wait for confirmation.
   On confirmation: assign current user (see **Assignment Routine**).

4. **Type gate** — check task type:
   - If type is `epic` → say: "This is a `type:epic` — it needs to be broken into features, then tasks. Run `kha:scoping` on it." STOP.
   - If type is `feature` → apply **Feature Advancement Rule** (see below). STOP after applying.
   - Proceed only for `type:task` or `type:bug`.

5. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`

6. Extract:
   - Acceptance criteria from `[kha:scoping]` comment or `[kha:design:context]` comment
   - Architecture context from `[kha:design:context]` comment
   - If neither exists → ask: "I couldn't find scoping or design comments on this task. Should I proceed without them, or should it go back to design first?" Wait for answer before proceeding.

7. Create branch from `develop` — always branch from `develop`, never from `main`:
   ```bash
   git checkout develop && git pull origin develop
   git checkout -b task/<task-id>-<kebab-title>
   ```

8. Move task to `IN DEVELOPMENT`. Start time tracking (see **Time Tracking**).

9. **TDD loop** — for each acceptance criterion from `[kha:scoping]` or `[kha:design:context]`, in order:
   - a. **Red** — write a failing test that exercises exactly that criterion. Run it to confirm it fails for the right reason (not a setup error).
   - b. Commit: `test(<task-id>): <what it tests>`
   - c. **Green** — implement the minimum code to make the test pass. Read `[kha:design:context]` file hints first; follow existing codebase patterns.
   - d. Run all tests to confirm the new test passes and nothing regresses.
   - e. Commit: `feat(<task-id>): <what was implemented>` (use `fix(<task-id>):` if task type is `bug`)
   - f. If implementation required a structural decision not covered by `[kha:design:context]` → state the decision and ask for confirmation before proceeding.
   - g. If a test cannot be made green after a genuine implementation attempt → report the blocker with the failing test name and error, leave the task in `IN DEVELOPMENT`, and stop. Do not move to `IN REVIEW` with failing tests.

10. **Refactor pass** (optional) — after all criteria are green, clean up duplication or clarity issues introduced during the loop. Run all tests again. If anything was refactored:
    Commit: `refactor(<task-id>): <what was cleaned up>`

11. Add comment to the ClickUp task:
    ```
    [kha:develop]
    branch: task/<task-id>-<kebab-title>
    criteria implemented:
    - <criterion> → <test name>
    - <criterion> → <test name>
    notes: <architectural decisions made during implementation, or omit this line>
    ```

12. Move task to `IN REVIEW`. Stop time tracking (see **Time Tracking**).

## Feature Advancement Rule

When a `type:feature` is encountered, do not process it as a regular task. Instead:

1. Check for a `[kha:design]` comment. If none → say: "This feature hasn't been designed yet — run `kha:design` on it." STOP.
2. Extract child task IDs from the `child tasks:` line in `[kha:design]`.
3. Fetch the current status of each child task via `mcp__clickup__clickup_get_task`.
4. Find the **minimum child status** using pipeline order:
   `TRIAGE < BACKLOG < SCOPING < IN DESIGN < READY FOR DEVELOPMENT < IN DEVELOPMENT < IN REVIEW < TESTING < SHIPPED`
5. If minimum child status > parent's current status:
   - Move parent to minimum child status via `mcp__clickup__clickup_update_task`.
   - Add ClickUp comment: `[kha:auto] parent advanced to [status] — reflects minimum status among [N] children ([list of child IDs and their statuses]).`
   - Report: "Feature **[title]** (`[id]`) advanced: [old status] → [new status]."
6. If minimum child status ≤ parent's current status:
   - Report which children are at or behind the parent's current status.
   - Say: "Feature cannot advance — child tasks have not yet reached this phase." STOP without changing status.

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
| Branch | task/[id]-[kebab-title] |
| Tests | [N] passing |
| Status | → IN REVIEW |
