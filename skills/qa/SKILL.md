---
name: kha:qa
description: Use when testing tasks in TESTING status. Writes and runs automated tests (unit, integration, Playwright e2e), handles manual fallback, and moves to SHIPPED on full pass.
---

# kha: QA

Processes tasks in `TESTING` status. Writes and runs automated tests tied to acceptance criteria. For criteria that genuinely cannot be automated (confirmed with human), creates a manual checklist and moves the task to `MANUAL TESTING`. Moves to `SHIPPED` on full automated pass.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm status names; check if `MANUAL TESTING` status exists in the list
3. Read the project's test setup: test runner, test directory structure, existing test files, Playwright config (if any)

## MANUAL TESTING Status

If `MANUAL TESTING` status does not exist in the ClickUp list, create it once using `mcp__clickup__clickup_update_list` or equivalent before moving any task there. Use orderindex after `TESTING`, color `#f4c430`. Reuse from then on — do not recreate.

## No Silent Assumptions

Never classify a test criterion as "not automatable" without:
1. Explaining exactly why it cannot be automated
2. Getting explicit human confirmation

## Steps

1. Fetch all tasks in `TESTING` using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in TESTING" and stop
3. For each task:
   - a. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`
   - b. Extract from comment thread:
     - Acceptance criteria from `[kha:scoping]`
     - Architecture context from `[kha:design]`
     - Review summary from `[kha:code-review]`
   - c. **Assess automability per criterion:**
     - Unit/integration testable → write test in the appropriate test file
     - UI/user-flow testable → write Playwright e2e test (`page.getByRole`, `page.getByLabel` — never CSS class selectors)
     - Unclear → stop and confirm: "I couldn't find a reliable way to automate '<criterion>' — here's why: <reason>. Do you agree this needs manual testing?" Wait for explicit agreement before classifying as manual.
   - d. **Write automated tests** (before running):
     - One test per criterion — test names describe the behavior: `test('resets password when valid token provided')`
     - Unit tests: test one function/method, mock all external dependencies (DB, network, time)
     - Integration tests: test module boundaries, mock only external services
     - Playwright e2e: test full user flows against the running app
     - Each test has exactly one assertion focus — split tests that check multiple behaviors
   - e. **Ask before committing:** "I've written <N> tests covering <criteria list>. Here's a summary: <test names>. Should I commit them?" Wait for confirmation.
   - f. On confirmation: commit test files
     ```bash
     git add <test files>
     git commit -m "test(<task-id>): add automated tests for <task title>"
     ```
   - g. **Run all automated tests** and report pass/fail per test mapped to its criterion
   - h. **Decision:**
     - All criteria covered by passing tests → move to `SHIPPED`:
       ```
       [kha:qa] result: passed
       automated: <N> tests, all passing
       coverage:
       - <criterion> → <test name>
       ```
     - Some criteria need manual testing (confirmed with human) → ensure `MANUAL TESTING` status exists, move task there:
       ```
       [kha:qa] result: manual required
       automated: <N> tests, all passing
       manual checklist:
       - [ ] <specific step: what to do and what to verify>
       - [ ] <specific step: what to do and what to verify>
       ```
     - Automated tests fail → stay in `TESTING`:
       ```
       [kha:qa] result: failed
       failing tests:
       - <test name> — covers: <criterion> — error: <error message>
       ```

## Test Writing Guidelines

- Unit tests: one function/method per test, mock all external dependencies
- Integration tests: test module boundaries, mock only external services (DB, APIs, file system)
- Playwright e2e: use `page.getByRole()`, `page.getByLabel()`, `page.getByText()` — never CSS class or ID selectors
- Test names use plain English describing behavior: `test('shows error when email not found')`
- Each test checks one thing — split tests that assert on multiple independent behaviors

## Output

Summary table after all tasks are processed:

| Task | Automated | Manual | Result |
|------|-----------|--------|--------|
| Password reset | 4 tests pass | — | → SHIPPED |
| Admin data export | 2 tests pass | 1 (visual layout) | → MANUAL TESTING |
| Auth middleware | 3 tests fail | — | stays TESTING |
