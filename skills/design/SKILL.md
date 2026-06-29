---
name: kha:design
description: Use when designing tasks in IN DESIGN status. Processes type:feature only — breaks into type:task children, defines architecture, and moves to READY FOR DEVELOPMENT. Processes ONE task per invocation.
---

# kha: Design

> **ONE TASK PER INVOCATION.** Iterate the ordered list; skip `type:task` and `type:bug` items (auto-advancing them); present the first `type:feature` to the user. If the user declines, present the next. Feature advancement is reported in the JSON — no manual check needed.

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

1. Fetch the first IN DESIGN task (timer starts automatically; Feature Advancement Rule applied internally):
   ```bash
   result=$($KHA next "in design" --list <LIST_ID> --pipeline "$PIPELINE")
   ```
   - If `task` is null → report "No items in IN DESIGN" and stop.
   - Report any `advanced_features` from the JSON before continuing.

2. **Type routing** (check `task.task_type` from JSON):
   - **`type:epic`** → `$KHA cancel <task.id>`, say: "This is a `type:epic` — break it into features first. Run `kha:scoping`." STOP.
   - **`type:task` or `type:bug`** → leaf node, auto-advance:
     ```bash
     $KHA update <task.id> --status "ready for development" --stop-timer
     ```
     Report: "Task `[id]` auto-advanced to READY FOR DEVELOPMENT."
     Fetch next: `result=$($KHA next "in design" --list <LIST_ID> --pipeline "$PIPELINE" --skip <all,seen,ids>)`
     Loop back to step 2.
   - **`type:feature`** → candidate found, proceed to selection loop.

3. **Selection loop:**
   - Present: "Found: **[task.name]** (ID: `[task.id]`). Process this task?"
   - **Confirmed** → assign user: `$KHA update <task.id> --assign`. Proceed to step 4.
   - **Declined** → `$KHA cancel <task.id>`, fetch next with `--skip`, loop back to step 2.

4. All context is in the JSON:
   - `task.description` — task content
   - `kha_blocks.scoping` — user-facing acceptance criteria and affected roles
   - If `kha_blocks.scoping` is absent → confirm: "There's no scoping comment on this task. Should I proceed with technical design only, or should it go back to scoping first?" Wait for answer.

5. **Analyze the codebase** — read relevant files, trace existing patterns around the feature area. Understand what already exists before proposing anything new.

6. **Architecture proposal** — always present before proceeding:
   - If changes fit existing patterns: "I'd implement this as `<X>` in `<file/module>` because `<Y>`. Does that match your expectations?"
   - If changes require new patterns: describe the new pattern, justify it, ask for confirmation
   - Wait for explicit agreement in both cases.

7. **Task breakdown** — after architecture is confirmed:
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
     - <implementation criterion — starts with verb>
     file hints: <relevant files or modules to look at>
     ```

8. If architecture or data flow is non-trivial → ask: "I'd like to create an architecture doc with diagrams and data models. Should I?" If agreed → create ClickUp doc (Mermaid diagrams, data models, API contracts), link in comment.

9. Finalize feature task:
   ```bash
   $KHA update <task.id> \
     --status "ready for development" \
     --comment "[kha:design]\narchitecture: <2-3 sentence summary>\nchild tasks: <id>, <id>, ..." \
     --stop-timer
   ```

10. Task complete. One invocation = one feature designed.

## Output

| Field | Value |
|-------|-------|
| Task | [title] ([id]) |
| Child Tasks | [N] created |
| Doc | [yes / no] |
| Status | → READY FOR DEVELOPMENT |
