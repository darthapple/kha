---
name: kha:develop
description: Use when developing tasks in READY FOR DEVELOPMENT status. Iterates the ordered list, skips features that can't advance, presents the first valid type:task or type:bug to the user, creates a branch, implements using TDD with small commits, and moves to IN REVIEW. Processes ONE task per invocation.
---

# kha: Develop

> **ONE TASK PER INVOCATION.** The binary handles Feature Advancement Rule and ordering. Present the first valid `type:task` or `type:bug` to the user. Feature advancement events are reported in the JSON response.

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

1. Fetch the first actionable task (Feature Advancement Rule applied internally; timer starts automatically):
   ```bash
   result=$($KHA next "ready for development" --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   - If `task` is null → report "No items in READY FOR DEVELOPMENT" and stop.
   - Report any `advanced_features` from the JSON.

2. **Type gate** (check `task.task_type`):
   - **`type:epic`** → `$KHA cancel <task.id>`, say: "This is a `type:epic` — break it into features first. Run `kha:scoping`." STOP.
   - **`type:feature`** → should not occur (binary skips these); if it does: `$KHA cancel <task.id>`, fetch next with `--skip`, loop.
   - **`type:task` or `type:bug`** → proceed to selection loop.

3. **Selection loop:**
   - Present: task name, ID, one-line summary from `kha_blocks["design:context"].scope` if present.
   - Ask: "Work on this task?"
   - **Confirmed** → assign user: `$KHA update <task.id> --assign`. Proceed to step 4.
   - **Declined** → `$KHA cancel <task.id>`, fetch next:
     ```bash
     result=$($KHA next "ready for development" --list <LIST_ID> --pipeline "$PIPELINE" --skip <all,seen,ids>)
     ```
     Loop back to step 2.

4. Extract from JSON:
   - Acceptance criteria from `kha_blocks["design:context"].acceptance_criteria` or `kha_blocks.scoping.acceptance_criteria`
   - Architecture context from `kha_blocks["design:context"].architecture` and `file_hints`
   - If neither exists → ask: "I couldn't find scoping or design comments on this task. Should I proceed without them, or should it go back to design first?" Wait before proceeding.

5. Create branch from `develop`:
   ```bash
   git checkout develop && git pull origin develop
   git checkout -b task/<task.id>-<kebab-title>
   ```

6. Move task to IN DEVELOPMENT (timer already running from step 1):
   ```bash
   $KHA update <task.id> --status "in development"
   ```

7. **TDD loop** — for each acceptance criterion, in order:
   - a. **Red** — write a failing test. Run it to confirm it fails for the right reason.
   - b. Commit: `test(<task.id>): <what it tests>`
   - c. **Green** — implement minimum code to pass. Follow `file_hints` and existing patterns.
   - d. Run all tests to confirm new test passes and nothing regresses.
   - e. Commit: `feat(<task.id>): <what was implemented>` (use `fix(...)` for bugs)
   - f. If a structural decision arose not covered by design context → state it and ask for confirmation before proceeding.
   - g. If a test cannot be made green → report blocker with failing test name and error. Leave task in IN DEVELOPMENT. Stop. Do not move to IN REVIEW with failing tests.

8. **Refactor pass** (optional) — clean up duplication or clarity issues. Run all tests again.
   Commit: `refactor(<task.id>): <what was cleaned up>`

9. Finalize:
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
