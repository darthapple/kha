---
name: kha:review
description: Use when reviewing tasks in IN REVIEW status. Reviews implementation against acceptance criteria, stack best practices, and security. Moves to TESTING on pass, stays IN REVIEW with findings on fail.
---

# kha: Review

Reviews tasks in `IN REVIEW` status. Evaluates the implementation against acceptance criteria from scoping, architecture decisions from design, stack best practices, and security. Moves to `TESTING` on pass or stays in `IN REVIEW` with specific, actionable findings on fail.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
3. Read the project's stack files (`package.json`, framework config, language version) to determine which best practices and security rules apply

## Platform Setup

Run once per session, cache `$KHA`:
```bash
_OS=$(uname -s 2>/dev/null || echo "Windows")
case "$_OS" in
  Darwin) [ "$(uname -m)" = "arm64" ] && KHA=~/.kha/kha-darwin-arm64 || KHA=~/.kha/kha-darwin-amd64 ;;
  Linux)  KHA=~/.kha/kha-linux-amd64 ;;
  *)      KHA="$APPDATA/kha/kha.exe" ;;
esac
```

## Steps

1. Fetch the first IN REVIEW task (Feature Advancement Rule applied internally; timer starts automatically):
   ```bash
   result=$($KHA next "in review" --list <LIST_ID>)
   ```
   - If `task` is null → report "No items in IN REVIEW" and stop.
   - Report any `advanced_features` from the JSON.

2. **Type gate**: if `task.task_type` is `feature` → the binary already handled advancement; this case means nothing was advanced. `$KHA cancel <task.id>` and stop.

3. **Selection loop:**
   - Present: "Found: **[task.name]** (ID: `[task.id]`). Review this task?"
   - **Confirmed** → assign user: `$KHA update <task.id> --assign`. Proceed to step 4.
   - **Declined** → `$KHA cancel <task.id>`, fetch next:
     ```bash
     result=$($KHA next "in review" --list <LIST_ID> --skip <all,seen,ids>)
     ```
     Loop back to step 2.

4. All context is in the JSON:
   - `kha_blocks.scoping` — acceptance criteria
   - `kha_blocks.design` — architecture decisions
   - If neither exists → ask: "I couldn't find scoping or design comments. Should I proceed without acceptance criteria, or re-scope first?"

5. Read the current git diff:
   ```bash
   git diff develop...HEAD
   ```

6. **Review Layer 1 — Acceptance criteria:** For each criterion in `kha_blocks.scoping`, evaluate whether the implementation satisfies it. Cite file and line. Mark ✅ or ❌.

7. **Review Layer 2 — Best practices:** Check for idiomatic patterns, clarity, naming, dead code, unnecessary duplication. Cite file and line.

8. **Review Layer 3 — Security:** Check for OWASP Top 10 risks relevant to the stack:
   - Injection (SQL, command, template)
   - Broken authentication / session management
   - Sensitive data exposure (secrets in code, unencrypted storage)
   - Broken access control
   - XSS (if frontend)
   - Insecure deserialization
   - Input validation gaps
   - Stack-specific vulnerabilities (prototype pollution in JS, SSRF in server code, etc.)
   Cite file and line for every issue.

9. **Decision:**
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
