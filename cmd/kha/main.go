package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/darthapple/kha/internal/clickup"
	"github.com/darthapple/kha/internal/config"
	"github.com/darthapple/kha/internal/pipeline"
)

// NextResult is the JSON payload returned by `kha next`.
type NextResult struct {
	Task             *TaskSummary            `json:"task"`
	Message          string                  `json:"message,omitempty"`
	Comments         []clickup.Comment       `json:"comments,omitempty"`
	KhaBlocks        map[string]map[string]any `json:"kha_blocks,omitempty"`
	CurrentUser      *clickup.Member         `json:"current_user,omitempty"`
	AdvancedFeatures []AdvancedFeature       `json:"advanced_features,omitempty"`
}

type TaskSummary struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Status      string          `json:"status"`
	TaskType    string          `json:"task_type"`
	Description string          `json:"description"`
	URL         string          `json:"url"`
	Assignees   []clickup.Member `json:"assignees"`
}

type AdvancedFeature struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

// UpdateResult is returned by `kha update`.
type UpdateResult struct {
	TaskID  string   `json:"task_id"`
	Actions []string `json:"actions"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fatal(err)
	}
	client := clickup.NewClient(cfg.APIKey)

	switch os.Args[1] {
	case "next":
		runNext(client, cfg, os.Args[2:])
	case "update":
		runUpdate(client, os.Args[2:])
	case "cancel":
		runCancel(client, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

// ── kha next <status> --list <id> [--pipeline s1,s2,...] [--skip id1,id2] ───

func runNext(client *clickup.Client, cfg *config.Config, args []string) {
	fs := flag.NewFlagSet("next", flag.ExitOnError)
	listID := fs.String("list", "", "ClickUp list ID (required)")
	pipelineFlag := fs.String("pipeline", "", "comma-separated pipeline status order, low→high")
	skipCSV := fs.String("skip", "", "comma-separated task IDs to skip")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fatalf("usage: kha next <status> --list <id> [--pipeline s1,s2,...] [--skip id1,id2]")
	}
	status := strings.Join(fs.Args(), " ")
	if *listID == "" {
		fatalf("--list is required")
	}

	// build pipeline order: flag overrides config default
	pipelineStatuses := cfg.Pipeline
	if *pipelineFlag != "" {
		pipelineStatuses = splitPipeline(*pipelineFlag)
	}
	order := pipeline.NewOrder(pipelineStatuses)

	skip := parseCSV(*skipCSV)

	tasks, err := client.ListTasks(*listID, status)
	if err != nil {
		fatal(err)
	}
	if len(tasks) == 0 {
		printJSON(NextResult{Message: "No items in " + status})
		return
	}

	sorted := pipeline.SortHierarchical(tasks)

	user, err := client.GetCurrentUser()
	if err != nil {
		fatal(err)
	}

	// resolve team ID: take from first task
	teamID := sorted[0].TeamID

	var advanced []AdvancedFeature
	var chosen *clickup.TaskWithOrder

	for i := range sorted {
		t := &sorted[i]

		// skip explicitly excluded IDs
		if skip[t.ID] {
			continue
		}

		// get comments for this task to determine type when task_type is ambiguous
		taskComments, err := client.GetComments(t.ID)
		if err != nil {
			fatal(err)
		}
		blocks := pipeline.ParseKhaBlocks(taskComments)
		typeName := pipeline.TaskTypeName(t.RawType.String(), blocks)

		switch typeName {
		case "epic":
			// epics are not actionable here
			continue

		case "feature":
			oldStatus := t.Status.Status
			newStatus, err := pipeline.AdvanceFeature(
				t,
				order,
				func(id string) (*clickup.TaskWithOrder, error) { return client.GetTask(id) },
				func(id, s string) error {
					return client.UpdateTask(id, map[string]any{"status": s})
				},
				func(id, text string) error { return client.AddComment(id, text) },
			)
			if err != nil {
				fatal(err)
			}
			if newStatus != "" {
				advanced = append(advanced, AdvancedFeature{
					ID: t.ID, Name: t.Name,
					OldStatus: oldStatus, NewStatus: newStatus,
				})
			}
			continue // features are never the returned task

		default:
			chosen = t
			chosenComments := taskComments
			chosenBlocks := blocks

			// start timer on chosen task
			if teamID != "" {
				if err := client.StartTimer(teamID, chosen.ID); err != nil {
					// non-fatal: timer failure should not block the skill
					fmt.Fprintf(os.Stderr, "warning: could not start timer: %v\n", err)
				}
			}

			assignees := make([]clickup.Member, len(chosen.Assignees))
			copy(assignees, chosen.Assignees)

			printJSON(NextResult{
				Task: &TaskSummary{
					ID:          chosen.ID,
					Name:        chosen.Name,
					Status:      chosen.Status.Status,
					TaskType:    typeName,
					Description: chosen.Description,
					URL:         chosen.URL,
					Assignees:   assignees,
				},
				Comments:         chosenComments,
				KhaBlocks:        chosenBlocks,
				CurrentUser:      user,
				AdvancedFeatures: advanced,
			})
			return
		}
	}

	// No actionable task found
	printJSON(NextResult{
		Message:          "No actionable items in " + status,
		AdvancedFeatures: advanced,
		CurrentUser:      user,
	})
}

// ── kha update <task-id> [flags] ────────────────────────────────────────────

func runUpdate(client *clickup.Client, args []string) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	statusFlag := fs.String("status", "", "move task to this status")
	commentFlag := fs.String("comment", "", "add comment (use \\n for newlines)")
	fileFlag := fs.String("file", "", "attach file at this path")
	assignFlag := fs.Bool("assign", false, "assign current user")
	stopTimer := fs.Bool("stop-timer", false, "stop active time entry")
	startTimer := fs.Bool("start-timer", false, "start time entry")
	fs.Parse(args)

	if fs.NArg() < 1 {
		fatalf("usage: kha update <task-id> [--status X] [--comment text] [--file path] [--assign] [--stop-timer] [--start-timer]")
	}
	taskID := fs.Arg(0)

	result := UpdateResult{TaskID: taskID}

	// fetch task once for team ID and existing assignees
	var task *clickup.TaskWithOrder
	needTask := *assignFlag || *stopTimer || *startTimer

	if needTask {
		t, err := client.GetTask(taskID)
		if err != nil {
			fatal(err)
		}
		task = t
	}

	if *statusFlag != "" {
		if err := client.UpdateTask(taskID, map[string]any{"status": *statusFlag}); err != nil {
			fatal(err)
		}
		result.Actions = append(result.Actions, "status → "+*statusFlag)
	}

	if *assignFlag {
		user, err := client.GetCurrentUser()
		if err != nil {
			fatal(err)
		}
		if err := client.AssignUser(taskID, user.ID, task.Assignees); err != nil {
			fatal(err)
		}
		result.Actions = append(result.Actions, fmt.Sprintf("assigned user %d", user.ID))
	}

	if *commentFlag != "" {
		text := clickup.ResolveNewlines(*commentFlag)
		if err := client.AddComment(taskID, text); err != nil {
			fatal(err)
		}
		result.Actions = append(result.Actions, "comment added")
	}

	if *fileFlag != "" {
		if err := client.UploadAttachment(taskID, *fileFlag); err != nil {
			fatal(err)
		}
		result.Actions = append(result.Actions, "file attached: "+*fileFlag)
	}

	if *stopTimer {
		if err := client.StopTimer(task.TeamID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not stop timer: %v\n", err)
		} else {
			result.Actions = append(result.Actions, "timer stopped")
		}
	}

	if *startTimer {
		if err := client.StartTimer(task.TeamID, taskID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not start timer: %v\n", err)
		} else {
			result.Actions = append(result.Actions, "timer started")
		}
	}

	if len(result.Actions) == 0 {
		result.Actions = []string{"no-op"}
	}
	printJSON(result)
}

// ── kha cancel <task-id> ────────────────────────────────────────────────────

func runCancel(client *clickup.Client, args []string) {
	fs := flag.NewFlagSet("cancel", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		fatalf("usage: kha cancel <task-id>")
	}
	taskID := fs.Arg(0)

	task, err := client.GetTask(taskID)
	if err != nil {
		fatal(err)
	}
	if err := client.StopTimer(task.TeamID); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not stop timer: %v\n", err)
	}

	printJSON(map[string]string{"task_id": taskID, "action": "timer stopped"})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func usage() {
	fmt.Fprintln(os.Stderr, `kha — ClickUp integration for kha skills

Usage:
  kha next <status> --list <id> [--pipeline s1,s2,...] [--skip id1,id2]
  kha update <task-id> [--status X] [--comment text] [--file path] [--assign] [--stop-timer] [--start-timer]
  kha cancel <task-id>`)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fatal(err)
	}
}

func parseCSV(s string) map[string]bool {
	m := make(map[string]bool)
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			m[v] = true
		}
	}
	return m
}

// splitPipeline splits a comma-separated pipeline string into an ordered slice.
func splitPipeline(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
