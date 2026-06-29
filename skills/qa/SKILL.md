---
name: kha:qa
description: Use when testing tasks in TESTING status. Writes and runs automated tests (unit, integration, Playwright e2e), handles manual fallback, and moves to SHIPPED on full pass. Processes ONE task per invocation.
---

# kha: QA

> **ONE TASK PER INVOCATION.** The binary handles Feature Advancement Rule and ordering. Present the first valid task to the user. Feature advancement events are reported in the JSON response.

Processes one task in `TESTING` status. Writes and runs automated tests tied to acceptance criteria. For criteria that genuinely cannot be automated (confirmed with human), creates a manual checklist and moves to `MANUAL TESTING`. Moves to `SHIPPED` on full automated pass.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm status names; check if `MANUAL TESTING` status exists
3. Read the project's test setup: test runner, test directory structure, existing test files, Playwright config (if any)

## MANUAL TESTING Status

If `MANUAL TESTING` status does not exist in the ClickUp list, create it once using `mcp__clickup__clickup_update_list` before moving any task there. Use orderindex after `TESTING`, color `#f4c430`. Reuse — do not recreate.

## No Silent Assumptions

Never classify a criterion as "not automatable" without:
1. Explaining exactly why it cannot be automated
2. Getting explicit human confirmation

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

1. Fetch the first TESTING task (Feature Advancement Rule applied internally; timer starts automatically):
   ```bash
   result=$($KHA next testing --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   - If `task` is null → report "No items in TESTING" and stop.
   - Report any `advanced_features` from the JSON.

2. **Selection loop:**
   - Present: "Found: **[task.name]** (ID: `[task.id]`). Process this task?"
   - **Confirmed** → assign user: `$KHA update <task.id> --assign`. Proceed to step 3.
   - **Declined** → `$KHA cancel <task.id>`, fetch next:
     ```bash
     result=$($KHA next testing --list <LIST_ID> --pipeline "$PIPELINE" --skip <all,seen,ids>)
     ```
     Loop back to step 2.

3. Extract from JSON:
   - For `type:task`: implementation-scope criteria from `kha_blocks["design:context"]` or `kha_blocks.scoping`
   - For `type:feature`: user-facing criteria from `kha_blocks.scoping`
   - Architecture context from `kha_blocks.design`
   - Review summary from `kha_blocks.review`

4. **Assess automability per criterion:**
   - `type:task` criteria → unit tests or integration tests
   - `type:feature` criteria → Playwright e2e tests
   - Unclear → confirm with human: "I couldn't find a reliable way to automate `<criterion>` — here's why: `<reason>`. Do you agree this needs manual testing?" Wait for explicit agreement.

5. **Write automated tests** (before running):
   - One test per criterion — names describe behavior: `test('resets password when valid token provided')`
   - Unit tests: one function/method, mock all external dependencies
   - Integration tests: test module boundaries, mock only external services
   - Playwright e2e: `page.getByRole`, `page.getByLabel`, `page.getByText` — never CSS class or ID selectors
   - Each test has exactly one assertion focus

6. **Ask before committing:** "I've written `<N>` tests covering `<criteria list>`. Here's a summary: `<test names>`. Should I commit them?" Wait for confirmation.

7. Commit test files:
   ```bash
   git add <test files>
   git commit -m "test(<task.id>): add automated tests for <task title>"
   ```

8. **Run all automated tests** and report pass/fail per test mapped to its criterion.

9. **Decision:**

   **Rule: fail overrides manual.** Any failing automated test keeps task in TESTING regardless of manual criteria.

   - **All passing** → merge and ship:
     ```bash
     git checkout develop && git pull origin develop
     git merge --no-ff task/<task.id>-<kebab-title> -m "Merge task/<task.id>-<kebab-title> into develop"
     git push origin develop
     git branch -d task/<task.id>-<kebab-title>
     gh pr create --base main --head develop --title "Release: <task title>" --body "Merges develop into main shipping task <task.id>: <task title>."
     ```
     If a `develop → main` PR is already open, skip creation and add the PR URL to the comment.
     ```bash
     $KHA update <task.id> \
       --status shipped \
       --comment "[kha:qa]\nresult: passed\nautomated: <N> tests, all passing\ncoverage:\n- <criterion> → <test name>\npr: <PR URL>" \
       --stop-timer
     ```

   - **Manual testing required** (all automated pass, some criteria confirmed as manual):
     ```bash
     $KHA update <task.id> \
       --status "manual testing" \
       --comment "[kha:qa]\nresult: manual required\nautomated: <N> tests, all passing\nmanual checklist:\n- [ ] <specific step: what to do and what to verify>" \
       --stop-timer
     ```

   - **Automated tests fail:**
     ```bash
     $KHA update <task.id> \
       --comment "[kha:qa]\nresult: failed\nfailing tests:\n- <test name> — covers: <criterion> — error: <error>" \
       --stop-timer
     ```
     Task stays in TESTING.

## Test Writing Guidelines

- Unit tests: one function/method per test, mock all external dependencies
- Integration tests: test module boundaries, mock only external services (DB, APIs, file system)
- Playwright e2e: `page.getByRole()`, `page.getByLabel()`, `page.getByText()` — never CSS class or ID selectors
- Test names use plain English: `test('shows error when email not found')`
- Each test checks one thing — split tests that assert on multiple independent behaviors

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Automated Tests | [N] tests |
| Manual | [yes / no] |
| Result | → [SHIPPED / MANUAL TESTING / stays TESTING] |
