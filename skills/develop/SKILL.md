---
name: kha:develop
description: Use when developing tasks in READY FOR DEVELOPMENT status. Picks the top task by column order, creates a branch, implements using TDD with small commits, and moves to IN REVIEW.
---

# kha: Develop

Picks the top task in `READY FOR DEVELOPMENT`, implements it using TDD with small commits on a dedicated branch, and moves to `IN REVIEW`.

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

1. Fetch all tasks in `READY FOR DEVELOPMENT` using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in READY FOR DEVELOPMENT" and stop
   Sort the returned tasks by their `orderindex` field ascending before selecting — this reflects the position within the status column (top to bottom). Never reorder by age, priority, or any other field.
3. Present the top task: title, ID, and one-line summary from its `[kha:design:context]` comment (if present). Ask: "Work on this task?" Wait for confirmation.
4. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`
5. Extract:
   - Acceptance criteria from `[kha:scoping]` comment
   - Architecture context from `[kha:design:context]` comment
   - If neither exists → ask: "I couldn't find scoping or design comments on this task. Should I proceed without them, or should it go back to design first?" Wait for answer before proceeding.
6. Create branch: `git checkout -b task/<task-id>-<kebab-title>` from current main
7. Move task to `IN DEVELOPMENT`
8. **TDD loop** — for each acceptance criterion from `[kha:scoping]`, in order:
   - a. **Red** — write a failing test that exercises exactly that criterion. Run it to confirm it fails for the right reason (not a setup error).
   - b. Commit: `test(<task-id>): <what it tests>`
   - c. **Green** — implement the minimum code to make the test pass. Read `[kha:design:context]` file hints first; follow existing codebase patterns.
   - d. Run all tests to confirm the new test passes and nothing regresses.
   - e. Commit: `feat(<task-id>): <what was implemented>` (use `fix(<task-id>):` if task type is `bug`)
   - f. If implementation required a structural decision not covered by `[kha:design:context]` → state the decision and ask for confirmation before proceeding.
   - g. If a test cannot be made green after a genuine implementation attempt → report the blocker with the failing test name and error, leave the task in `IN DEVELOPMENT`, and stop. Do not move to `IN REVIEW` with failing tests.
9. **Refactor pass** (optional) — after all criteria are green, clean up duplication or clarity issues introduced during the loop. Run all tests again. If anything was refactored:
   Commit: `refactor(<task-id>): <what was cleaned up>`
10. Add comment to the ClickUp task:
    ```
    [kha:develop]
    branch: task/<task-id>-<kebab-title>
    criteria implemented:
    - <criterion> → <test name>
    - <criterion> → <test name>
    notes: <architectural decisions made during implementation, or omit this line>
    ```
11. Move task to `IN REVIEW`

## Output

Summary table after the task is processed:

| Task | Branch | Tests | Status |
|------|--------|-------|--------|
| Add CSV export | task/abc123-add-csv-export | 3 passing | → IN REVIEW |
