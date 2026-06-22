---
name: kha:code-review
description: Use when reviewing tasks in IN REVIEW status. Reviews implementation against acceptance criteria, stack best practices, and security. Moves to TESTING on pass, stays IN REVIEW with findings on fail.
---

# kha: Code Review

Reviews tasks in `IN REVIEW` status. Evaluates the implementation against acceptance criteria from scoping, architecture decisions from design, stack best practices, and security. Moves to `TESTING` on pass or stays in `IN REVIEW` with specific, actionable findings on fail.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
   (Taxonomy doc not needed by this skill — no task classification performed)
3. Read the project's stack files (`package.json`, framework config, language version) to determine which best practices and security rules apply

## Steps

1. Fetch all tasks in `IN REVIEW` using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in IN REVIEW" and stop
3. For each task:
   - a. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`
   - b. Extract from comment thread:
     - Acceptance criteria from `[kha:scoping]` comment
     - Architecture decisions from `[kha:design]` comment
     - If neither comment exists → ask: "I couldn't find scoping or design comments on this task. Should I proceed without acceptance criteria, or should the task be re-scoped first?"
   - c. Read the current git diff (run `git diff HEAD` or `git diff main...HEAD` as appropriate) focusing on files relevant to the task
   - d. **Review Layer 1 — Acceptance criteria:** For each criterion in `[kha:scoping]`, evaluate whether the implementation satisfies it. Cite file and line for each finding. Mark ✅ or ❌.
   - e. **Review Layer 2 — Best practices:** Check for: idiomatic patterns for the detected stack, code clarity and naming, dead code, unnecessary duplication, structural issues. Cite file and line.
   - f. **Review Layer 3 — Security:** Check for OWASP Top 10 risks relevant to the stack:
     - Injection (SQL, command, template)
     - Broken authentication / session management
     - Sensitive data exposure (secrets in code, unencrypted storage)
     - Broken access control
     - XSS (if frontend)
     - Insecure deserialization
     - Input validation gaps
     - Stack-specific vulnerabilities (e.g. prototype pollution in JS, SSRF in server code)
     Cite file and line for every issue.
   - g. **Decision:**
     - All criteria ✅ and no blocking best practice or security issue → move to `TESTING`:
       ```
       [kha:code-review] result: approved
       criteria: all met
       notes: <non-blocking observations, or omit this line>
       ```
     - Any criterion ❌ or any blocking issue → stay in `IN REVIEW`:
       ```
       [kha:code-review] result: changes requested
       criteria:
       - ✅ <criterion text>
       - ❌ <criterion text> — <file>:<line> — <what is missing or wrong>
       security:
       - <file>:<line> — <vulnerability type> — <explanation and fix> (omit section if none)
       practices:
       - <file>:<line> — <issue> — <explanation and fix> (omit section if none)
       ```

## Finding Quality Rules

- Every finding cites file and line — never say "somewhere in the code"
- Every finding explains exactly what to change and why — no vague guidance
- Non-blocking observations (style, minor improvements) go in `notes`, never in `criteria` or `security`
- Security findings always include the risk and the concrete fix
- A best-practice finding is **blocking** when it introduces maintenance risk (architectural mismatch, dead execution paths, or logic duplication that diverges from the same logic elsewhere). Style, naming, and cosmetic issues are never blocking — record them in `notes`.

## Output

Summary table after all tasks are processed:

| Task | Result | Criteria Issues | Security Issues |
|------|--------|-----------------|-----------------|
| Password reset | approved | 0 | 0 |
| Auth refactor | changes requested | 1 | 2 |
