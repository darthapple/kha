---
name: kha:design
description: Use when designing tasks in IN DESIGN status. Processes type:feature only — breaks into type:task children, defines architecture, and moves to READY FOR DEVELOPMENT. Processes ONE task per invocation.
---

# kha: Design

> **ONE TASK PER INVOCATION.** Pick the first task only (top of column by orderindex).
> After completing it, STOP. Never loop to the next task.
> Batch processing is forbidden — the user must re-invoke the skill for each task.

Processes one `type:feature` task in `IN DESIGN` status. Analyzes the codebase, defines architecture, breaks the feature into `type:task` children, and moves to `READY FOR DEVELOPMENT`.

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
2. If none → report "No items in IN DESIGN" and stop.
   Sort the returned tasks by their `orderindex` field ascending. Select `tasks[0]` only.

3. Present the task to the user: "Found: **[title]** (ID: `[id]`). Process this task?" Wait for confirmation.

4. **Type gate** — fetch task type:
   - If type is `task` or `bug` → move the task to `READY FOR DEVELOPMENT`, then say: "This is a `type:[task|bug]` — leaf node, no design breakdown needed. Moved to READY FOR DEVELOPMENT. Run `kha:develop` on it." STOP.
   - If type is `epic` → say: "This is a `type:epic` — epics should be broken into features first. Run `kha:scoping` on it." STOP.
   - Proceed only for `type:feature`.

5. Fetch full task details: `mcp__clickup__clickup_get_task` (include `description`) + `mcp__clickup__clickup_get_task_comments`

6. Extract business context from `[kha:scoping]` comment (user-facing acceptance criteria, affected roles).
   If no `[kha:scoping]` comment is present → stop and confirm: "There's no scoping comment on this task. Should I proceed with technical design only, or should it go back to scoping first?" Wait for answer before continuing.

7. **Analyze the codebase** — read relevant files, trace existing patterns around the feature area. Understand what already exists before proposing anything new.

8. **Architecture proposal** — always present your analysis before proceeding:
   - If changes fit existing patterns: "I'd implement this as `<X>` in `<file/module>` because `<Y>`. Does that match your expectations?"
   - If changes require new patterns or structural adjustments: describe the new pattern, justify it, and ask for confirmation before proceeding
   - Wait for explicit agreement in both cases

9. **Task breakdown** — after architecture is confirmed:
   - Propose a numbered list of independent `type:task` children. Each entry: title, one-paragraph description, which part of the architecture it covers, and the **implementation-scope acceptance criteria** for that task
   - Each child task must deliver concrete, testable value on its own
   - Ask: "I'd break this into these tasks — does this look right before I create them?" Wait for answer.
   - On agreement: create each as a `type:task` with `mcp__clickup__clickup_create_task`:
     `parent_id` = current task ID, `status` = `READY FOR DEVELOPMENT`, `list_id` from AGENTS.md, `task_type` = `Task`
   - Add `[kha:design:context]` comment to each child task:
     ```
     [kha:design:context]
     parent feature: <feature title> (<feature id>)
     architecture: <relevant architecture context for this task>
     scope: <exactly what this task covers>
     acceptance criteria:
     - <implementation criterion — starts with verb>
     - <implementation criterion — starts with verb>
     file hints: <relevant files or modules to look at>
     ```

10. If architecture or data flow is non-trivial → ask: "I'd like to create an architecture doc with diagrams and data models. Should I?" Wait for answer.
    - If agreed → create ClickUp doc (use Mermaid for diagrams, include data models and API contracts if relevant), link in comment

11. Add comment to feature task:
    ```
    [kha:design]
    architecture: <2-3 sentence summary of the approach>
    child tasks: <id>, <id>, ...
    doc: <url if created, else omit this line>
    ```

12. Move feature task to `READY FOR DEVELOPMENT`

13. **STOP.** Do not process any remaining tasks in the queue.
    One invocation = one task. The user must re-invoke `kha:design` for the next task.

## Output

Report for the single processed task:

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Child Tasks | [N] created |
| Doc | [yes / no] |
| Status | → READY FOR DEVELOPMENT |
