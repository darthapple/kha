---
name: kha:develop
description: Use when developing tasks in READY FOR DEVELOPMENT status. Iterates the ordered list, presents the first valid type:task or type:bug to the user, creates a branch, implements using TDD with small commits, and moves to IN REVIEW. Processes ONE task per invocation.
---

# kha: Develop

> **ONE TASK PER INVOCATION.** Fetch all READY FOR DEVELOPMENT tasks once, iterate locally, process one. Do not call `$KHA next` more than once.

Finds the first actionable `type:task` or `type:bug` in `READY FOR DEVELOPMENT`, implements it using TDD with small commits on a dedicated branch, and moves to `IN REVIEW`.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc ID
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm status names

## No Silent Assumptions

Never assume architecture, file structure, or implementation scope. When anything is ambiguous:
1. State what you observed and why it is uncertain
2. Present your suggestion with reasoning
3. Wait for explicit agreement before acting

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

> **Call `$KHA next` exactly once.** It returns all tasks in READY FOR DEVELOPMENT. Iterate `result.tasks` locally — never call `$KHA next` again during this session.

1. Fetch all READY FOR DEVELOPMENT tasks:
   ```bash
   result=$($KHA next "ready for development" --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   - If `result.tasks` is empty → report "No items in READY FOR DEVELOPMENT" and stop.
   - Report any `result.advanced_features`.

2. **Selection loop** — iterate `result.tasks` from index 0:
   - If all tasks exhausted → report "No actionable tasks remaining" and stop.
   - Check `task.task_type`:
     - **`epic`** → say "This is an epic — run `kha:scoping` first." Skip to next in array.
     - **`feature`** → say "This is a feature — run `kha:design` first." Skip to next in array.
     - **`task` or `bug`** → present: "[task.name] (`[task.task_type]`) — [one-line summary from `task.kha_blocks["design:context"].scope` if present]. Work on this?"
   - **Declined** → advance to next in the array. Loop.
   - **Confirmed** → assign user and start timer:
     ```bash
     $KHA update <task.id> --start-timer --assign
     ```
     Proceed to step 3.

3. Extract from the task object:
   - Acceptance criteria from `task.kha_blocks["design:context"].acceptance_criteria` or `task.kha_blocks.scoping.acceptance_criteria`
   - Architecture context from `task.kha_blocks["design:context"].architecture` and `.file_hints`
   - If neither exists → ask: "I couldn't find scoping or design comments on this task. Should I proceed without them, or should it go back to design first?" Wait before proceeding.

4. Move task to IN DEVELOPMENT (timer already running from step 2):
   ```bash
   $KHA update <task.id> --status "in development"
   ```

5. Create branch from `develop`:
   ```bash
   git checkout develop && git pull origin develop
   git checkout -b task/<task.id>-<kebab-title>
   ```

6. **TDD loop** — for each acceptance criterion, in order:
   - a. **Red** — write a failing test. Run it to confirm it fails for the right reason.
   - b. Commit: `test(<task.id>): <what it tests>`
   - c. **Green** — implement minimum code to pass. Follow `file_hints` and existing patterns.
   - d. Run all tests to confirm new test passes and nothing regresses.
   - e. Commit: `feat(<task.id>): <what was implemented>` (use `fix(...)` for bugs)
   - f. If a structural decision arose not covered by design context → state it and ask for confirmation before proceeding.
   - g. If a test cannot be made green → report blocker with failing test name and error. Leave task in IN DEVELOPMENT. Stop. Do not move to IN REVIEW with failing tests.

7. **Refactor pass** (optional) — clean up duplication or clarity issues. Run all tests again.
   Commit: `refactor(<task.id>): <what was cleaned up>`

8. Finalize:
   ```bash
   $KHA update <task.id> \
     --status "in review" \
     --comment "[kha:develop]\nbranch: task/<task.id>-<kebab-title>\ncriteria implemented:\n- <criterion> → <test name>\nnotes: <decisions, or omit>" \
     --stop-timer
   ```

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Branch | task/[id]-[kebab-title] |
| Tests | [N] passing |
| Status | → IN REVIEW |
