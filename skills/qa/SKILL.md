---
name: kha:qa
description: Use when testing tasks in TESTING status. Writes and runs automated tests, handles manual fallback, and moves to SHIPPED. Processes ONE task per invocation.
---

# kha: QA

> **ONE TASK PER INVOCATION.** Call `$KHA next` exactly once. All task data — description, comments, kha_blocks — is in the returned JSON. Never call `$KHA next` again. Never fetch tasks or comments separately.

Processes one task in `TESTING` status. Writes and runs automated tests tied to acceptance criteria. Moves to `SHIPPED` on full pass.

## Context

Read `AGENTS.md` once → note `list_id` and the pipeline order (the `→`-separated statuses in the Pipeline section). Read test setup: runner, directory structure, existing test files, Playwright config.

Do NOT read the ClickUp Pipeline or Taxonomy docs — they are not needed.

## MANUAL TESTING Status

If `MANUAL TESTING` does not exist in the list, create it once via `mcp__clickup__clickup_update_list` (orderindex after TESTING, color `#f4c430`). Reuse — do not recreate.

## No Silent Assumptions

Never classify a criterion as "not automatable" without explaining why and getting explicit human confirmation.

## Steps

**Step 1 — Fetch all TESTING tasks (run this entire block as one bash command):**

```bash
KHA="$HOME/.kha/kha"; [ -f .env.local ] && source .env.local
"$KHA" next testing --list <LIST_ID> --pipeline "<PIPELINE>"
```

Replace `<LIST_ID>` with the list ID from `AGENTS.md`. Replace `<PIPELINE>` with the pipeline from `AGENTS.md` (the `→`-separated statuses, lowercased, comma-separated).

**If this command exits with an error → report the exact error text and stop. Do NOT retry.**

**The response JSON has this exact shape:**
```json
{
  "tasks": [
    {
      "id": "86e22abc",
      "name": "Task title",
      "status": "testing",
      "task_type": "task",
      "description": "...",
      "url": "https://app.clickup.com/t/86e22abc",
      "assignees": [{ "id": 123, "email": "user@example.com" }],
      "comments": [...],
      "kha_blocks": {
        "scoping": { "acceptance_criteria": ["..."] },
        "design:context": { "scope": "...", "acceptance_criteria": ["..."] },
        "design": { "architecture": "..." },
        "review": { "result": "approved" }
      }
    }
  ],
  "current_user": { "id": 123, "email": "user@example.com" },
  "advanced_features": [],
  "message": "(only present when tasks array is empty)"
}
```

If `tasks` is empty → report `message` + any `advanced_features` and stop.

---

**From this point, work entirely from the JSON above. Do NOT call `$KHA next` again.**

- Iterate `tasks` by index in your reasoning. No CLI calls on decline.
- For `type:task`: use criteria from `tasks[i].kha_blocks["design:context"]` or `.scoping`
- For `type:feature`: use criteria from `tasks[i].kha_blocks.scoping`

---

**Step 2 — Selection loop:**

- Start at `tasks[0]`. Present: "Found: **[name]**. Process this task?"
- **Declined** → move to `tasks[1]`, etc. No CLI call.
- **All exhausted** → report "No tasks remaining in TESTING" and stop.
- **Confirmed** → assign and start timer:
  ```bash
  "$KHA" update <task.id> --start-timer --assign
  ```

**Step 3 — Assess automability per criterion:**
- `type:task` → unit/integration tests
- `type:feature` → Playwright e2e tests
- Unclear → confirm with human: "I can't reliably automate `<criterion>` because `<reason>`. Do you agree this needs manual testing?" Wait for explicit agreement.

**Step 4 — Write automated tests** (before running):
- One test per criterion; names describe behavior: `test('resets password when valid token provided')`
- Unit: one function/method, mock all external dependencies
- Integration: test module boundaries, mock only external services
- Playwright: `page.getByRole`, `page.getByLabel`, `page.getByText` — never CSS class or ID selectors
- Each test has exactly one assertion focus

**Step 5 — Ask before committing:**
"I've written `<N>` tests covering `<criteria>`. Commit them?" Wait for confirmation.

**Step 6 — Commit:**
```bash
git add <test files>
git commit -m "test(<task.id>): add automated tests for <task title>"
```

**Step 7 — Run all tests** and report pass/fail per test mapped to its criterion.

**Step 8 — Decision** (fail overrides manual):

All passing → merge and ship:
```bash
git checkout develop && git pull origin develop
git merge --no-ff task/<task.id>-<kebab-title> -m "Merge task/<task.id>-<kebab-title> into develop"
git push origin develop
git branch -d task/<task.id>-<kebab-title>
gh pr create --base main --head develop --title "Release: <task title>" --body "Merges develop into main shipping task <task.id>: <task title>."
```
If a `develop → main` PR is already open, skip creation and add the URL to the comment instead.
```bash
"$KHA" update <task.id> \
  --status shipped \
  --comment "[kha:qa]\nresult: passed\nautomated: <N> tests\ncoverage:\n- <criterion> → <test name>\npr: <PR URL>" \
  --stop-timer
```

Manual testing required (automated pass, some criteria confirmed manual):
```bash
"$KHA" update <task.id> \
  --status "manual testing" \
  --comment "[kha:qa]\nresult: manual required\nautomated: <N> tests\nmanual checklist:\n- [ ] <step: what to do and verify>" \
  --stop-timer
```

Automated tests fail:
```bash
"$KHA" update <task.id> \
  --comment "[kha:qa]\nresult: failed\nfailing tests:\n- <test name> — covers: <criterion> — error: <error>" \
  --stop-timer
```

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Automated Tests | [N] |
| Manual | [yes / no] |
| Result | → [SHIPPED / MANUAL TESTING / stays TESTING] |
