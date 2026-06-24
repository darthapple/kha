---
name: kha:qa
description: Use when testing tasks in TESTING status. Writes and runs automated tests (unit, integration, Playwright e2e), handles manual fallback, and moves to SHIPPED on full pass. Processes ONE task per invocation.
---

# kha: QA

> **ONE TASK PER INVOCATION.** Pick the first task only (top of column by orderindex).
> After completing it, STOP. Never loop to the next task.
> Batch processing is forbidden — the user must re-invoke the skill for each task.

Processes one task in `TESTING` status. Writes and runs automated tests tied to acceptance criteria. For criteria that genuinely cannot be automated (confirmed with human), creates a manual checklist and moves to `MANUAL TESTING`. Moves to `SHIPPED` on full automated pass.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm status names; check if `MANUAL TESTING` status exists in the list
   (Taxonomy doc not needed by this skill — no task classification performed)
3. Read the project's test setup: test runner, test directory structure, existing test files, Playwright config (if any)

## MANUAL TESTING Status

If `MANUAL TESTING` status does not exist in the ClickUp list, create it once using `mcp__clickup__clickup_update_list` or equivalent before moving any task there. Use orderindex after `TESTING`, color `#f4c430`. Reuse from then on — do not recreate.

## No Silent Assumptions

Never classify a test criterion as "not automatable" without:
1. Explaining exactly why it cannot be automated
2. Getting explicit human confirmation

## Steps

1. Fetch all tasks in `TESTING` using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in TESTING" and stop.
   Sort the returned tasks by their `orderindex` field ascending. Select `tasks[0]` only.

3. Present the task to the user: "Found: **[title]** (ID: `[id]`). Process this task?" Wait for confirmation.
   On confirmation: assign current user (see **Assignment Routine**). Start time tracking (see **Time Tracking**).

4. Fetch full task details: `mcp__clickup__clickup_get_task` (include `description`) + `mcp__clickup__clickup_get_task_comments`

5. Extract from comment thread:
   - **For `type:task`:** implementation-scope acceptance criteria from `[kha:scoping]` or `[kha:design:context]`
   - **For `type:feature`:** user-facing acceptance criteria from `[kha:scoping]`
   - Architecture context from `[kha:design]`
   - Review summary from `[kha:review]`

6. **Assess automability per criterion:**
   - `type:task` criteria (implementation-scope) → unit tests or integration tests
   - `type:feature` criteria (user-facing flows) → Playwright e2e tests
   - Unclear → stop and confirm: "I couldn't find a reliable way to automate '<criterion>' — here's why: <reason>. Do you agree this needs manual testing?" Wait for explicit agreement before classifying as manual.

7. **Write automated tests** (before running):
   - One test per criterion — test names describe the behavior: `test('resets password when valid token provided')`
   - Unit tests: test one function/method, mock all external dependencies (DB, network, time)
   - Integration tests: test module boundaries, mock only external services
   - Playwright e2e: test full user flows against the running app; use `page.getByRole`, `page.getByLabel`, `page.getByText` — never CSS class or ID selectors
   - Each test has exactly one assertion focus — split tests that check multiple behaviors

8. **Ask before committing:** "I've written <N> tests covering <criteria list>. Here's a summary: <test names>. Should I commit them?" Wait for confirmation.

9. On confirmation: commit test files
   ```bash
   git add <test files>
   git commit -m "test(<task-id>): add automated tests for <task title>"
   ```
   **Note:** Tests are committed before running. If the run fails, the commit remains — this is intentional: the test files capture the test intent and are useful for the developer to inspect even when failing.

10. **Run all automated tests** and report pass/fail per test mapped to its criterion

11. **Decision:**
    - **Rule: fail overrides manual.** If any automated test fails, the task stays in `TESTING` regardless of manual criteria. Fix failing tests first, then re-run. Manual criteria are only evaluated when all automated tests pass.
    - All criteria covered by passing tests → merge task branch into `develop`, open a PR to merge `develop` into `main`, then move to `SHIPPED`. Stop time tracking (see **Time Tracking**):
      ```bash
      git checkout develop && git pull origin develop
      git merge --no-ff task/<task-id>-<kebab-title> -m "Merge task/<task-id>-<kebab-title> into develop"
      git push origin develop
      git branch -d task/<task-id>-<kebab-title>
      ```
      Then open a PR from `develop` into `main`:
      ```bash
      gh pr create --base main --head develop --title "Release: <task title>" --body "Merges develop into main shipping task <task-id>: <task title>."
      ```
      If a `develop → main` PR is already open, skip creation and add the PR URL to the comment instead.
      Then add comment and move status:
      ```
      [kha:qa] result: passed
      automated: <N> tests, all passing
      coverage:
      - <criterion> → <test name>
      pr: <develop→main PR URL>
      ```
    - Some criteria need manual testing (confirmed with human) → ensure `MANUAL TESTING` status exists, move task there. Stop time tracking (see **Time Tracking**):
      ```
      [kha:qa] result: manual required
      automated: <N> tests, all passing
      manual checklist:
      - [ ] <specific step: what to do and what to verify>
      - [ ] <specific step: what to do and what to verify>
      ```
    - Automated tests fail → stay in `TESTING`. Stop time tracking (see **Time Tracking**):
      ```
      [kha:qa] result: failed
      failing tests:
      - <test name> — covers: <criterion> — error: <error message>
      ```

12. **STOP.** Do not process any remaining tasks in the queue.
    One invocation = one task. The user must re-invoke `kha:qa` for the next task.

## Assignment Routine

When starting work on a task, ensure the current user is assigned:
1. Call `mcp__clickup__clickup_get_workspace_members` and find the member with email `fernando.adriano@kheperi.com.br` — note their user ID. (Look up once per session and reuse.)
2. Check the task's existing `assignees` from the fetched task details.
3. If current user is **not** in the list: call `mcp__clickup__clickup_update_task` with `assignees` = all existing assignee IDs + current user ID.
4. If already assigned: skip.

## Time Tracking

**Start:** Call `mcp__clickup__clickup_start_time_tracking` with `task_id`. ClickUp automatically stops any previously active entry.

**Stop:** Call `mcp__clickup__clickup_stop_time_tracking`.

## Test Writing Guidelines

- Unit tests: one function/method per test, mock all external dependencies
- Integration tests: test module boundaries, mock only external services (DB, APIs, file system)
- Playwright e2e: use `page.getByRole()`, `page.getByLabel()`, `page.getByText()` — never CSS class or ID selectors
- Test names use plain English describing behavior: `test('shows error when email not found')`
- Each test checks one thing — split tests that assert on multiple independent behaviors

## Output

Report for the single processed task:

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Automated Tests | [N] tests |
| Manual | [yes / no] |
| Result | → [SHIPPED / MANUAL TESTING / stays TESTING] |
