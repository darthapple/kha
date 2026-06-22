---
name: kha:scoping
description: Use when scoping tasks in BACKLOG status. Performs business analysis, detects epics, writes acceptance criteria, and moves to IN DESIGN.
---

# kha: Scoping

Processes all tasks in `BACKLOG` status. Classifies intent, detects epics, performs business analysis, writes acceptance criteria, and moves each task to `IN DESIGN`.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc ID, taxonomy doc ID
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
3. Read Taxonomy doc (`_Config` space, doc ID: `2kza2py5-537`) → label rules

## No Silent Assumptions

Never assume intent, scope, or classification. When anything is ambiguous:
1. State what you observed and why it is uncertain
2. Present your suggestion with reasoning
3. Wait for explicit agreement before acting

The only actions allowed without confirmation: reading data, moving to the doing state (`SCOPING`), and adding informational comments.

## Steps

1. Fetch all tasks in `BACKLOG` using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in BACKLOG" and stop
   Sort the returned tasks by their `orderindex` field ascending before processing — this reflects the position within the status column (top to bottom). Never reorder by age, priority, or any other field.
3. For each task:
   - a. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`
   - b. Move task to `SCOPING` status
   - c. **Classify intent** — business or technical?
     - Technical = refactor, devops, infra, or bug fix with no user-facing behavior change
     - If technical → stop and ask: "This looks technical rather than business-facing — I'd route it directly to IN DESIGN without business scoping. Agreed?"
     - If agreed → add comment `[kha:scoping] routed: non-business → IN DESIGN`, move to `IN DESIGN`, skip to next task
   - d. **Epic detection** — if `type:epic` label is set:
     - Propose a breakdown: present a numbered list of candidate child tasks (title + one-line description each)
     - Ask: "I'd break this epic into these tasks — does this look right before I create them?" Wait for answer.
     - On agreement: create each child as an independent task using `mcp__clickup__clickup_create_task` with `parent_id` = epic task ID, `status` = `BACKLOG`, `list_id` from AGENTS.md
     - Add `[kha:scoping:context]` comment to each child task:
       ```
       [kha:scoping:context]
       parent epic: <epic title> (<epic id>)
       business goal: <what this epic is trying to achieve>
       context: <relevant background the child task needs to be scoped independently>
       ```
     - The epic itself does NOT get acceptance criteria — those belong to the child tasks
     - Add comment to epic:
       ```
       [kha:scoping]
       type: epic
       epic: yes
       routed: business
       affected roles: N/A (see child tasks)
       child tasks: <id>, <id>, ...
       ```
     - Move epic to `IN DESIGN`, skip to next task
   - e. **Business analysis** (non-epic, business-routed tasks):
     - Write acceptance criteria: each criterion is user-facing, testable, unambiguous, and starts with a verb
     - Identify user roles affected by this task
     - If the UI interaction is non-trivial or scope is unclear → ask: "The interaction here is non-trivial — I'd like to create a wireframe/low-level design doc before proceeding. Should I?" Wait for answer.
     - If agreed → create a ClickUp doc with wireframes and flow description, link it in the comment
   - f. Add comment to task:
     ```
     [kha:scoping]
     type: <type>
     epic: no
     routed: business
     affected roles: <comma-separated list>
     acceptance criteria:
     - <criterion — starts with verb>
     - <criterion — starts with verb>
     doc: <url if created, else omit this line>
     ```
   - g. Move task to `IN DESIGN`

## Output

Summary table after all tasks are processed:

| Task | Routed | Epic | Status |
|------|--------|------|--------|
| Reset password flow | business | no | → IN DESIGN |
| Extract auth module | non-business | no | → IN DESIGN |
| User onboarding | business | yes | → IN DESIGN (3 child tasks) |
