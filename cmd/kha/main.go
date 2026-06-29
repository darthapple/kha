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

// TaskEntry is one item in the NextResult tasks array.
type TaskEntry struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Status      string                    `json:"status"`
	TaskType    string                    `json:"task_type"`
	Description string                    `json:"description"`
	URL         string                    `json:"url"`
	Assignees   []clickup.Member          `json:"assignees"`
	Comments    []clickup.Comment         `json:"comments"`
	KhaBlocks   map[string]map[string]any `json:"kha_blocks"`
}

// NextResult is the JSON payload returned by `kha next`.
type NextResult struct {
	Tasks            []TaskEntry       `json:"tasks"`
	Message          string            `json:"message,omitempty"`
	CurrentUser      *clickup.Member   `json:"current_user,omitempty"`
	AdvancedFeatures []AdvancedFeature `json:"advanced_features,omitempty"`
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

// ── kha next <status> --list <id> [--pipeline s1,s2,...] ────────────────────
//
// Returns ALL tasks in the given status as a sorted array. Features that are
// advanced by the Feature Advancement Rule are excluded (they moved status).
// Features that are NOT advanced are included. No timer is started.

func runNext(client *clickup.Client, cfg *config.Config, args []string) {
	if len(args) < 1 {
		fatalf("usage: kha next <status> --list <id> [--pipeline s1,s2,...]")
	}
	// Status is always the first arg; flags follow. Go's flag parser stops at
	// the first non-flag arg, so we must separate status from flags manually.
	status := args[0]

	fs := flag.NewFlagSet("next", flag.ExitOnError)
	listID := fs.String("list", "", "ClickUp list ID (required)")
	pipelineFlag := fs.String("pipeline", "", "comma-separated pipeline status order, low→high")
	fs.Parse(args[1:])

	if *listID == "" {
		fatalf("--list is required")
	}

	pipelineStatuses := cfg.Pipeline
	if *pipelineFlag != "" {
		pipelineStatuses = splitPipeline(*pipelineFlag)
	}
	order := pipeline.NewOrder(pipelineStatuses)

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

	var advanced []AdvancedFeature
	var entries []TaskEntry

	for i := range sorted {
		t := &sorted[i]

		taskComments, err := client.GetComments(t.ID)
		if err != nil {
			fatal(err)
		}
		blocks := pipeline.ParseKhaBlocks(taskComments)
		typeName := pipeline.TaskTypeName(t.RawType.String(), blocks)

		if typeName == "feature" {
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
				// Feature was advanced — it moved status, exclude from list
				advanced = append(advanced, AdvancedFeature{
					ID: t.ID, Name: t.Name,
					OldStatus: oldStatus, NewStatus: newStatus,
				})
				continue
			}
			// Feature not advanced — include it so the skill can handle it
		}

		assignees := make([]clickup.Member, len(t.Assignees))
		copy(assignees, t.Assignees)

		entries = append(entries, TaskEntry{
			ID:          t.ID,
			Name:        t.Name,
			Status:      t.Status.Status,
			TaskType:    typeName,
			Description: t.Description,
			URL:         t.URL,
			Assignees:   assignees,
			Comments:    taskComments,
			KhaBlocks:   blocks,
		})
	}

	if len(entries) == 0 {
		printJSON(NextResult{
			Message:          "No actionable items in " + status,
			AdvancedFeatures: advanced,
			CurrentUser:      user,
		})
		return
	}

	printJSON(NextResult{
		Tasks:            entries,
		CurrentUser:      user,
		AdvancedFeatures: advanced,
	})
}

// ── kha update <task-id> [flags] ────────────────────────────────────────────

func runUpdate(client *clickup.Client, args []string) {
	if len(args) < 1 {
		fatalf("usage: kha update <task-id> [--status X] [--comment text] [--file path] [--assign] [--stop-timer] [--start-timer]")
	}
	// Task ID is always the first positional arg; parse flags from the rest.
	// Go's flag package stops at the first non-flag token, so if we passed
	// the task ID through fs.Parse it would silently swallow all subsequent flags.
	taskID := args[0]

	fs := flag.NewFlagSet("update", flag.ExitOnError)
	statusFlag := fs.String("status", "", "move task to this status")
	commentFlag := fs.String("comment", "", "add comment (use \\n for newlines)")
	fileFlag := fs.String("file", "", "attach file at this path")
	assignFlag := fs.Bool("assign", false, "assign current user")
	stopTimer := fs.Bool("stop-timer", false, "stop active time entry")
	startTimer := fs.Bool("start-timer", false, "start time entry")
	fs.Parse(args[1:])

	result := UpdateResult{TaskID: taskID}

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
  kha next <status> --list <id> [--pipeline s1,s2,...]
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
