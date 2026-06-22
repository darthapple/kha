---
name: kha:design
description: Use when designing tasks in IN DESIGN status. Performs technical analysis, defines architecture, breaks into child tasks, and moves to READY FOR DEVELOPMENT.
---

# kha: Design

Processes all tasks in `IN DESIGN` status. Analyzes the codebase, defines or adjusts architecture, breaks the task into development-ready child tasks, and moves to `READY FOR DEVELOPMENT`.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc ID, taxonomy doc ID
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
3. Read Taxonomy doc (`_Config` space, doc ID: `2kza2py5-537`) → label rules

## No Silent Assumptions

Never assume architecture, file structure, or task scope. When anything is ambiguous:
1. State what you observed and why it is uncertain
2. Present your suggestion with reasoning
3. Wait for explicit agreement before acting

The only actions allowed without confirmation: reading data and adding informational comments. Moving to `READY FOR DEVELOPMENT` requires all decisions above to have been confirmed.

## Steps

1. Fetch all tasks in `IN DESIGN` using `mcp__clickup__clickup_filter_tasks`
2. If none → report "No items in IN DESIGN" and stop
3. For each task:
   - a. Fetch full task details: `mcp__clickup__clickup_get_task` + `mcp__clickup__clickup_get_task_comments`
   - b. Extract business context from `[kha:scoping]` or `[kha:scoping:context]` comment if present
     If neither comment is present → stop and confirm: "There's no scoping comment on this task. Should I proceed with technical design only, or should it go back to scoping first?" Wait for answer before continuing.
   - c. **Analyze the codebase** — read relevant files, trace existing patterns around the feature area. The goal is to understand what already exists before proposing anything new.
   - d. **Architecture proposal** — always present your analysis before proceeding:
     - If changes fit existing patterns: "I'd implement this as <X> in <file/module> because <Y>. Does that match your expectations?"
     - If changes require new patterns or structural adjustments: describe the new pattern, justify it, and ask for confirmation before proceeding
     - Wait for explicit agreement in both cases
   - e. **Task breakdown** — after architecture is confirmed:
     - Propose a numbered list of independent child tasks. Each entry: title, one-paragraph description, which part of the architecture it covers
     - Each child task must deliver concrete, testable value on its own
     - Ask: "I'd break this into these tasks — does this look right before I create them?" Wait for answer.
     - On agreement: create each as an independent task with `mcp__clickup__clickup_create_task`: `parent_id` = current task ID, `status` = `BACKLOG`, `list_id` from AGENTS.md
     - Add `[kha:design:context]` comment to each child task:
       ```
       [kha:design:context]
       parent task: <parent title> (<parent id>)
       architecture: <relevant architecture context for this child>
       scope: <exactly what this child task covers>
       file hints: <relevant files or modules to look at>
       ```
   - f. If architecture or data flow is non-trivial → ask: "I'd like to create an architecture doc with diagrams and data models. Should I?" Wait for answer.
     - If agreed → create ClickUp doc (use Mermaid for diagrams, include data models and API contracts if relevant), link in comment
   - g. Add comment to task:
     ```
     [kha:design]
     architecture: <2-3 sentence summary of the approach>
     child tasks: <id>, <id>, ...
     doc: <url if created, else omit this line>
     ```
   - h. Move task to `READY FOR DEVELOPMENT`

## Output

Summary table after all tasks are processed:

| Task | Child Tasks | Doc | Status |
|------|-------------|-----|--------|
| Password reset flow | 3 created | yes | → READY FOR DEVELOPMENT |
| Auth module refactor | 2 created | no | → READY FOR DEVELOPMENT |
