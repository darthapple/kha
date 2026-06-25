---
name: kha:design
description: Use when designing tasks in IN DESIGN status. Processes type:feature only — breaks into type:task children, defines architecture, and moves to READY FOR DEVELOPMENT. Processes ONE task per invocation.
---

# kha: Design

> **ONE TASK PER INVOCATION.** Iterate the ordered list; skip `type:task` and `type:bug` items (auto-advancing them); present the first `type:feature` to the user. If the user declines, present the next. Process only one task per invocation — declining is selection, not processing.

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

1. Fetch tasks in `IN DESIGN` in column order using curl (MCP strips `orderindex`). Get list ID from `AGENTS.md`, API key from `.env.local`:
   ```bash
   source .env.local && curl -s "https://api.clickup.com/api/v2/list/<LIST_ID>/task?statuses[]=in%20design&subtasks=true" -H "Authorization: $CLICKUP_API_KEY"
   ```
   Build column order hierarchically: (1) separate top-level tasks (`parent` is null) from subtasks; (2) sort top-level tasks by `orderindex` ascending; (3) for each top-level task in order, insert its direct subtasks sorted by `orderindex` ascending immediately after it — this mirrors ClickUp's visual grouping where subtasks appear under their parent.
2. If response contains no tasks → report "No items in IN DESIGN" and stop.

3. **Selection loop** — iterate the ordered list from position 0:
   - If list is exhausted → report "No features to design in IN DESIGN" and stop.
   - **`type:epic`** → say: "This is a `type:epic` — break it into features first. Run `kha:scoping`." STOP.
   - **`type:task` or `type:bug`** → leaf node, no design needed: move to `READY FOR DEVELOPMENT`, report "Task `[id]` auto-advanced to READY FOR DEVELOPMENT", skip silently, advance position, continue loop.
   - **`type:feature`** → candidate found:
     - Present: "Found: **[title]** (ID: `[id]`). Process this task?"
     - Confirmed → assign current user (see **Assignment Routine**), start time tracking (see **Time Tracking**), break loop, proceed to step 4.
     - Declined → advance position, continue loop.

4. Fetch full task details: `mcp__clickup__clickup_get_task` (include `description`) + `mcp__clickup__clickup_get_task_comments`

5. Extract business context from `[kha:scoping]` comment (user-facing acceptance criteria, affected roles).
   If no `[kha:scoping]` comment is present → stop and confirm: "There's no scoping comment on this task. Should I proceed with technical design only, or should it go back to scoping first?" Wait for answer before continuing.

6. **Analyze the codebase** — read relevant files, trace existing patterns around the feature area. Understand what already exists before proposing anything new.

7. **Architecture proposal** — always present your analysis before proceeding:
   - If changes fit existing patterns: "I'd implement this as `<X>` in `<file/module>` because `<Y>`. Does that match your expectations?"
   - If changes require new patterns or structural adjustments: describe the new pattern, justify it, and ask for confirmation before proceeding
   - Wait for explicit agreement in both cases

8. **Task breakdown** — after architecture is confirmed:
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

9. If architecture or data flow is non-trivial → ask: "I'd like to create an architecture doc with diagrams and data models. Should I?" Wait for answer.
   - If agreed → create ClickUp doc (use Mermaid for diagrams, include data models and API contracts if relevant), link in comment

10. Add comment to feature task:
    ```
    [kha:design]
    architecture: <2-3 sentence summary of the approach>
    child tasks: <id>, <id>, ...
    doc: <url if created, else omit this line>
    ```

11. Move feature task to `READY FOR DEVELOPMENT`. Stop time tracking (see **Time Tracking**).

12. Task complete. One invocation = one feature designed.

## Assignment Routine

When starting work on a task, ensure the current user is assigned:
1. Call `mcp__clickup__clickup_get_workspace_members` and find the member with email `fernando.adriano@kheperi.com.br` — note their user ID. (Look up once per session and reuse.)
2. Check the task's existing `assignees` from the fetched task details.
3. If current user is **not** in the list: call `mcp__clickup__clickup_update_task` with `assignees` = all existing assignee IDs + current user ID.
4. If already assigned: skip.

## Time Tracking

**Start:** Call `mcp__clickup__clickup_start_time_tracking` with `task_id`. ClickUp automatically stops any previously active entry.

**Stop:** Call `mcp__clickup__clickup_stop_time_tracking`.

## Output

Report for the single processed task:

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Child Tasks | [N] created |
| Doc | [yes / no] |
| Status | → READY FOR DEVELOPMENT |
