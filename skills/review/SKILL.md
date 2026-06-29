---
name: kha:review
description: Use when reviewing tasks in IN REVIEW status. Reviews against acceptance criteria, best practices, and security. Moves to TESTING on pass. Processes ONE task per invocation.
---

# kha: Review

> **ONE TASK PER INVOCATION.** Call `$KHA next` exactly once. All task data — description, comments, kha_blocks — is in the returned JSON. Never call `$KHA next` again. Never fetch tasks or comments separately.

Reviews one task in `IN REVIEW` status against acceptance criteria, best practices, and security. Moves to `TESTING` on pass.

## Context

1. Read `AGENTS.md` → find `list_id`
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → get exact status names in order
3. Read the project's stack files (`package.json`, framework config) to know which security rules apply

## Steps

**Step 1 — Fetch all IN REVIEW tasks (run this entire block as one bash command):**

```bash
_OS=$(uname -s 2>/dev/null || echo "Windows")
case "$_OS" in
  Darwin) [ "$(uname -m)" = "arm64" ] && KHA="$HOME/.kha/kha-darwin-arm64" || KHA="$HOME/.kha/kha-darwin-amd64" ;;
  Linux)  KHA="$HOME/.kha/kha-linux-amd64" ;;
  *)      KHA="$APPDATA/kha/kha.exe" ;;
esac
[ -f .env.local ] && source .env.local
"$KHA" next "in review" --list <LIST_ID> --pipeline "<PIPELINE>"
```

**The response JSON has this exact shape:**
```json
{
  "tasks": [
    {
      "id": "86e22abc",
      "name": "Task title",
      "status": "in review",
      "task_type": "task",
      "description": "...",
      "url": "https://app.clickup.com/t/86e22abc",
      "assignees": [{ "id": 123, "email": "user@example.com" }],
      "comments": [...],
      "kha_blocks": {
        "scoping": { "acceptance_criteria": ["..."] },
        "design": { "architecture": "..." },
        "design:context": { "scope": "...", "acceptance_criteria": ["..."] },
        "develop": { "branch": "task/...", "criteria_implemented": ["..."] }
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
- `tasks[i].kha_blocks.scoping` and `tasks[i].kha_blocks["design:context"]` have acceptance criteria.
- `tasks[i].kha_blocks.design` has architecture decisions.

---

**Step 2 — Selection loop:**

- Start at `tasks[0]`. Present: "Found: **[name]**. Review this task?"
- **Declined** → move to `tasks[1]`, etc. No CLI call.
- **All exhausted** → report "No tasks remaining in IN REVIEW" and stop.
- **Confirmed** → assign and start timer:
  ```bash
  "$KHA" update <task.id> --start-timer --assign
  ```

**Step 3 — Check context:**
- No scoping or design blocks → ask: "No acceptance criteria found. Proceed without them, or re-scope first?" Wait.

**Step 4 — Read the git diff:**
```bash
git diff develop...HEAD
```

**Step 5 — Review Layer 1: Acceptance criteria** — for each criterion in `kha_blocks.scoping` or `kha_blocks["design:context"]`. Cite file and line. Mark ✅ or ❌.

**Step 6 — Review Layer 2: Best practices** — idiomatic patterns, clarity, naming, dead code, duplication. Cite file and line.

**Step 7 — Review Layer 3: Security** — OWASP Top 10 relevant to the stack:
- Injection (SQL, command, template)
- Broken auth / session management
- Sensitive data exposure
- Broken access control
- XSS (frontend), insecure deserialization, input validation gaps
- Stack-specific (prototype pollution in JS, SSRF in server code)

**Step 8 — Decision:**

All criteria ✅ and no blocking issue:
```bash
"$KHA" update <task.id> \
  --status testing \
  --comment "[kha:review]\nresult: approved\ncriteria: all met\nnotes: <non-blocking or omit>" \
  --stop-timer
```

Any ❌ or blocking issue:
```bash
"$KHA" update <task.id> \
  --comment "[kha:review]\nresult: changes requested\ncriteria:\n- ✅ <criterion>\n- ❌ <criterion> — <file>:<line> — <what is wrong>\nsecurity:\n- <file>:<line> — <type> — <fix>" \
  --stop-timer
```
Task stays in IN REVIEW.

## Finding Quality Rules

- Every finding cites file and line
- Every finding explains exactly what to change and why
- A best-practice finding is **blocking** only when it introduces maintenance risk (architectural mismatch, dead execution paths, diverging logic). Style is never blocking.

## Output

| Task | Result | Criteria Issues | Security Issues |
|------|--------|-----------------|-----------------|
| [title] | [approved / changes requested] | [N] | [N] |
