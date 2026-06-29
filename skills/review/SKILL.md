---
name: kha:review
description: Use when reviewing tasks in IN REVIEW status. Reviews implementation against acceptance criteria, stack best practices, and security. Moves to TESTING on pass, stays IN REVIEW with findings on fail.
---

# kha: Review

> **ONE TASK PER INVOCATION.** Fetch all IN REVIEW tasks once, iterate locally, process one. Do not call `$KHA next` more than once.

Reviews tasks in `IN REVIEW` status. Evaluates the implementation against acceptance criteria from scoping, architecture decisions from design, stack best practices, and security. Moves to `TESTING` on pass or stays in `IN REVIEW` with specific, actionable findings on fail.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
3. Read the project's stack files (`package.json`, framework config, language version) to determine which best practices and security rules apply

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

> **Call `$KHA next` exactly once.** It returns all tasks in IN REVIEW. Iterate `result.tasks` locally — never call `$KHA next` again during this session.

1. Fetch all IN REVIEW tasks:
   ```bash
   result=$($KHA next "in review" --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   - If `result.tasks` is empty → report "No items in IN REVIEW" and stop.
   - Report any `result.advanced_features`.

2. **Selection loop** — iterate `result.tasks` from index 0:
   - If all tasks exhausted → report "No tasks remaining in IN REVIEW" and stop.
   - Present: "Found: **[task.name]** (ID: `[task.id]`). Review this task?"
   - **Declined** → advance to next in the array. Loop.
   - **Confirmed** → assign user and start timer:
     ```bash
     $KHA update <task.id> --start-timer --assign
     ```
     Proceed to step 3.

3. All context is already in the task object:
   - `task.kha_blocks.scoping` — acceptance criteria
   - `task.kha_blocks.design` — architecture decisions
   - `task.kha_blocks["design:context"]` — per-task scope and criteria
   - If neither scoping nor design blocks exist → ask: "I couldn't find scoping or design comments. Should I proceed without acceptance criteria, or re-scope first?"

4. Read the current git diff:
   ```bash
   git diff develop...HEAD
   ```

5. **Review Layer 1 — Acceptance criteria:** For each criterion in `kha_blocks.scoping` or `kha_blocks["design:context"]`, evaluate whether the implementation satisfies it. Cite file and line. Mark ✅ or ❌.

6. **Review Layer 2 — Best practices:** Check for idiomatic patterns, clarity, naming, dead code, unnecessary duplication. Cite file and line.

7. **Review Layer 3 — Security:** Check for OWASP Top 10 risks relevant to the stack:
   - Injection (SQL, command, template)
   - Broken authentication / session management
   - Sensitive data exposure (secrets in code, unencrypted storage)
   - Broken access control
   - XSS (if frontend)
   - Insecure deserialization
   - Input validation gaps
   - Stack-specific vulnerabilities (prototype pollution in JS, SSRF in server code, etc.)
   Cite file and line for every issue.

8. **Decision:**
   - All criteria ✅ and no blocking issue:
     ```bash
     $KHA update <task.id> \
       --status testing \
       --comment "[kha:review]\nresult: approved\ncriteria: all met\nnotes: <non-blocking, or omit>" \
       --stop-timer
     ```
   - Any criterion ❌ or blocking issue:
     ```bash
     $KHA update <task.id> \
       --comment "[kha:review]\nresult: changes requested\ncriteria:\n- ✅ <criterion>\n- ❌ <criterion> — <file>:<line> — <what is wrong>\nsecurity:\n- <file>:<line> — <type> — <fix>\npractices:\n- <file>:<line> — <issue> — <fix>" \
       --stop-timer
     ```
     Task stays in IN REVIEW.

## Finding Quality Rules

- Every finding cites file and line — never say "somewhere in the code"
- Every finding explains exactly what to change and why
- Non-blocking observations go in `notes`, never in `criteria` or `security`
- Security findings always include the risk and the concrete fix
- A best-practice finding is **blocking** when it introduces maintenance risk (architectural mismatch, dead execution paths, diverging duplicate logic). Style and cosmetic issues are never blocking.

## Output

| Task | Result | Criteria Issues | Security Issues |
|------|--------|-----------------|-----------------|
| [title] | [approved / changes requested] | [N] | [N] |
