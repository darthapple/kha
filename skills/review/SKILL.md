---
name: kha:review
description: Use when reviewing tasks in IN REVIEW status. Reviews implementation against acceptance criteria, stack best practices, and security. Moves to TESTING on pass, stays IN REVIEW with findings on fail.
---

# kha: Review

Reviews tasks in `IN REVIEW` status. Evaluates the implementation against acceptance criteria from scoping, architecture decisions from design, stack best practices, and security. Moves to `TESTING` on pass or stays in `IN REVIEW` with specific, actionable findings on fail.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc IDs
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
   (Taxonomy doc not needed by this skill — no task classification performed)
3. Read the project's stack files (`package.json`, framework config, language version) to determine which best practices and security rules apply

## Steps

1. Fetch tasks in `IN REVIEW` in column order using curl (MCP strips `orderindex`). Get list ID from `AGENTS.md`, API key from `.env.local`:
   ```bash
   source .env.local && curl -s "https://api.clickup.com/api/v2/list/<LIST_ID>/task?statuses[]=in%20review&subtasks=true" -H "Authorization: $CLICKUP_API_KEY"
   ```
   Sort the returned `tasks` array by `orderindex` ascending — this reflects column order (top to bottom). Never reorder by age, priority, or any other field.
2. If response contains no tasks → report "No items in IN REVIEW" and stop.
3. For each task:
   - a. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`. Assign current user (see **Assignment Routine**). Start time tracking (see **Time Tracking**).
   - a2. **Type gate:** If task type is `feature` → apply **Feature Advancement Rule** (see below). Stop time tracking. Skip to next task.
   - b. Extract from comment thread:
     - Acceptance criteria from `[kha:scoping]` comment
     - Architecture decisions from `[kha:design]` comment
     - If neither comment exists → ask: "I couldn't find scoping or design comments on this task. Should I proceed without acceptance criteria, or should the task be re-scoped first?"
   - c. Read the current git diff (run `git diff develop...HEAD`) focusing on files relevant to the task
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
     - All criteria ✅ and no blocking best practice or security issue → move to `TESTING`. Stop time tracking (see **Time Tracking**). Add comment:
       ```
       [kha:review] result: approved
       criteria: all met
       notes: <non-blocking observations, or omit this line>
       ```
     - Any criterion ❌ or any blocking issue → stay in `IN REVIEW`. Stop time tracking (see **Time Tracking**). Add comment:
       ```
       [kha:review] result: changes requested
       criteria:
       - ✅ <criterion text>
       - ❌ <criterion text> — <file>:<line> — <what is missing or wrong>
       security:
       - <file>:<line> — <vulnerability type> — <explanation and fix> (omit section if none)
       practices:
       - <file>:<line> — <issue> — <explanation and fix> (omit section if none)
       ```

## Feature Advancement Rule

When a `type:feature` is encountered, do not review it as a regular task. Instead:

1. Check for a `[kha:design]` comment. If none → say: "This feature hasn't been designed yet — run `kha:design` on it." STOP.
2. Extract child task IDs from the `child tasks:` line in `[kha:design]`.
3. Fetch the current status of each child task via `mcp__clickup__clickup_get_task`.
4. Find the **minimum child status** using pipeline order:
   `TRIAGE < BACKLOG < SCOPING < IN DESIGN < READY FOR DEVELOPMENT < IN DEVELOPMENT < IN REVIEW < TESTING < SHIPPED`
5. If minimum child status > parent's current status:
   - Move parent to minimum child status via `mcp__clickup__clickup_update_task`.
   - Add ClickUp comment: `[kha:auto] parent advanced to [status] — reflects minimum status among [N] children ([list of child IDs and their statuses]).`
   - Report: "Feature **[title]** (`[id]`) advanced: [old status] → [new status]."
6. If minimum child status ≤ parent's current status:
   - Report which children are at or behind the parent's current status.
   - Say: "Feature cannot advance — child tasks have not yet reached this phase." STOP without changing status.

## Assignment Routine

When starting work on a task, ensure the current user is assigned:
1. Call `mcp__clickup__clickup_get_workspace_members` and find the member with email `fernando.adriano@kheperi.com.br` — note their user ID. (Look up once per session and reuse.)
2. Check the task's existing `assignees` from the fetched task details.
3. If current user is **not** in the list: call `mcp__clickup__clickup_update_task` with `assignees` = all existing assignee IDs + current user ID.
4. If already assigned: skip.

## Time Tracking

**Start:** Call `mcp__clickup__clickup_start_time_tracking` with `task_id`. ClickUp automatically stops any previously active entry.

**Stop:** Call `mcp__clickup__clickup_stop_time_tracking`.

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
