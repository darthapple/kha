---
name: kha:design
description: Use when designing tasks in IN DESIGN status. Analyzes codebase, defines architecture, breaks features into type:task children. Also handles bugs and tasks that need design analysis. Processes ONE task per invocation.
---

# kha: Design

> **ONE TASK PER INVOCATION.** Fetch all IN DESIGN tasks once, iterate locally, process one. Do not call `$KHA next` more than once.

Processes one task in `IN DESIGN` status. Analyzes the codebase, defines architecture, and either breaks the task into children (for features) or defines implementation approach (for bugs and tasks), then moves to `READY FOR DEVELOPMENT`.

## Context

1. Read `AGENTS.md` → get list ID, pipeline doc ID, taxonomy doc ID
2. Read Pipeline doc (`_Config` space, doc ID: `2kza2py5-517`) → confirm current status names
3. Read Taxonomy doc (`_Config` space, doc ID: `2kza2py5-537`) → label rules

## No Silent Assumptions

Never assume architecture, file structure, or task scope. When anything is ambiguous:
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

> **Call `$KHA next` exactly once.** It returns all tasks in IN DESIGN. Iterate `result.tasks` locally — never call `$KHA next` again during this session.

1. Fetch all IN DESIGN tasks:
   ```bash
   result=$($KHA next "in design" --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   - If `result.tasks` is empty → report "No items in IN DESIGN" and stop.
   - Report any `result.advanced_features` before continuing.

2. **Selection loop** — iterate `result.tasks` from index 0:
   - If all tasks exhausted → report "No tasks to design" and stop.
   - Check `task.task_type`:
     - **`epic`** → say "This is a `type:epic` — break it into features first. Run `kha:scoping`." Skip to next in array.
     - **`feature`, `task`, `bug`** → all go through design. Present: "Found: **[task.name]** (`[task.task_type]`). Design this task?"
   - **Declined** → advance to next in the array. Loop.
   - **Confirmed** → assign user and start timer:
     ```bash
     $KHA update <task.id> --start-timer --assign
     ```
     Proceed to step 3.

3. All context is already in the task object:
   - `task.description` — task content
   - `task.kha_blocks.scoping` — user-facing acceptance criteria and affected roles
   - `task.kha_blocks["scoping:context"]` — parent epic context if applicable
   - If `task.kha_blocks.scoping` is absent → confirm: "There's no scoping comment on this task. Should I proceed with technical design only, or should it go back to scoping first?" Wait for answer.

4. **Analyze the codebase** — read relevant files, trace existing patterns around the task area. Understand what already exists before proposing anything new.

5. **Architecture proposal** — always present before proceeding:
   - If changes fit existing patterns: "I'd implement this as `<X>` in `<file/module>` because `<Y>`. Does that match your expectations?"
   - If changes require new patterns: describe the new pattern, justify it, ask for confirmation
   - Wait for explicit agreement in both cases.

6. **Route by task type:**

   ### type:feature
   - **Task breakdown** — after architecture is confirmed:
     - Propose a numbered list of independent `type:task` children (title, description, architecture coverage, acceptance criteria)
     - Ask: "I'd break this into these tasks — does this look right before I create them?" Wait for answer.
     - On agreement: create each via `mcp__clickup__clickup_create_task`:
       `parent_id` = current task ID, `status` = `READY FOR DEVELOPMENT`, `list_id` from AGENTS.md, `task_type` = `Task`
     - Add `[kha:design:context]` comment to each child via `mcp__clickup__clickup_create_comment`:
       ```
       [kha:design:context]
       parent feature: <feature title> (<feature id>)
       architecture: <relevant architecture context for this task>
       scope: <exactly what this task covers>
       acceptance criteria:
       - <implementation criterion — starts with verb>
       file hints: <relevant files or modules to look at>
       ```
   - If architecture or data flow is non-trivial → ask: "I'd like to create an architecture doc with diagrams and data models. Should I?" If agreed → create ClickUp doc, link in comment.
   - Finalize:
     ```bash
     $KHA update <task.id> \
       --status "ready for development" \
       --comment "[kha:design]\narchitecture: <2-3 sentence summary>\nchild tasks: <id>, <id>, ..." \
       --stop-timer
     ```

   ### type:task or type:bug
   - Define implementation approach: which files change, what the fix/implementation looks like, any edge cases
   - Add `[kha:design:context]` comment directly on this task via `mcp__clickup__clickup_create_comment`:
     ```
     [kha:design:context]
     architecture: <approach summary>
     scope: <what this task covers>
     acceptance criteria:
     - <implementation criterion — starts with verb>
     file hints: <relevant files to look at>
     ```
   - Finalize:
     ```bash
     $KHA update <task.id> \
       --status "ready for development" \
       --comment "[kha:design]\narchitecture: <2-3 sentence summary>" \
       --stop-timer
     ```

7. Task complete. One invocation = one task designed.

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Type | [feature / task / bug] |
| Child Tasks | [N created / N/A] |
| Status | → READY FOR DEVELOPMENT |
