package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/darthapple/kha/internal/clickup"
)

// Order maps normalized status name → numeric position (low = early in pipeline).
type Order struct {
	index map[string]int
	names []string // original casing, in order
}

func NewOrder(statuses []string) *Order {
	o := &Order{index: make(map[string]int), names: statuses}
	for i, s := range statuses {
		o.index[norm(s)] = i
	}
	return o
}

func (o *Order) Rank(status string) int {
	if r, ok := o.index[norm(status)]; ok {
		return r
	}
	return -1
}

func (o *Order) Name(rank int) string {
	if rank >= 0 && rank < len(o.names) {
		return o.names[rank]
	}
	return ""
}

func norm(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// SortHierarchical returns tasks in ClickUp visual order:
// top-level tasks sorted by orderindex, each followed immediately by their subtasks.
func SortHierarchical(tasks []clickup.TaskWithOrder) []clickup.TaskWithOrder {
	var top []clickup.TaskWithOrder
	subtaskMap := make(map[string][]clickup.TaskWithOrder)

	for _, t := range tasks {
		if t.Parent == nil || *t.Parent == "" {
			top = append(top, t)
		} else {
			subtaskMap[*t.Parent] = append(subtaskMap[*t.Parent], t)
		}
	}

	sort.Slice(top, func(i, j int) bool {
		return getOrderIndex(top[i]) < getOrderIndex(top[j])
	})

	var result []clickup.TaskWithOrder
	for _, t := range top {
		result = append(result, t)
		children := subtaskMap[t.ID]
		sort.Slice(children, func(i, j int) bool {
			return getOrderIndex(children[i]) < getOrderIndex(children[j])
		})
		result = append(result, children...)
	}
	return result
}

func getOrderIndex(t clickup.TaskWithOrder) float64 { return t.OrderIndex.F() }

// ParseKhaBlocks scans comments and extracts [kha:*] structured blocks.
// Returns a map of block-name → key/value pairs (values are string or []string).
func ParseKhaBlocks(comments []clickup.Comment) map[string]map[string]any {
	blocks := make(map[string]map[string]any)

	for _, c := range comments {
		lines := strings.Split(c.Text, "\n")
		var blockName string
		var current map[string]any
		var lastKey string
		var listAccum []string

		flushList := func() {
			if lastKey != "" && len(listAccum) > 0 {
				current[lastKey] = listAccum
				listAccum = nil
			}
		}

		for _, raw := range lines {
			line := strings.TrimSpace(raw)

			// New [kha:name] header
			if strings.HasPrefix(line, "[kha:") {
				end := strings.Index(line, "]")
				if end > 5 {
					// save previous block
					if blockName != "" {
						flushList()
						blocks[blockName] = current
					}
					blockName = line[5:end]
					current = make(map[string]any)
					lastKey = ""
					listAccum = nil
					continue
				}
			}

			if blockName == "" {
				continue
			}

			// List item continuation
			if strings.HasPrefix(line, "- ") {
				listAccum = append(listAccum, strings.TrimPrefix(line, "- "))
				continue
			}

			// Empty line: flush list
			if line == "" {
				flushList()
				lastKey = ""
				continue
			}

			// key: value line
			if idx := strings.Index(line, ": "); idx > 0 {
				flushList()
				lastKey = strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+2:])
				if val != "" {
					current[lastKey] = val
				}
				// value might be empty when the next lines are list items
				continue
			}
		}

		// flush last block
		if blockName != "" {
			flushList()
			blocks[blockName] = current
		}
	}

	return blocks
}

// TaskTypeName extracts a normalized task type string ("feature", "bug", "task", "epic")
// from the raw task_type field and optionally from kha:triage block as fallback.
func TaskTypeName(rawType string, khaBlocks map[string]map[string]any) string {
	t := norm(rawType)
	switch t {
	case "feature", "bug", "epic", "task", "milestone":
		return t
	}

	// fallback: check [kha:triage] comment block
	if triage, ok := khaBlocks["triage"]; ok {
		if tv, ok := triage["type"].(string); ok {
			return norm(tv)
		}
	}
	return "task"
}

// AdvanceFeature checks if a feature's children have collectively moved past the feature's
// current pipeline position, and if so advances the feature.
// Returns the new status name (empty string if no advance was needed).
func AdvanceFeature(
	feature *clickup.TaskWithOrder,
	order *Order,
	fetchTask func(id string) (*clickup.TaskWithOrder, error),
	updateStatus func(id, status string) error,
	addComment func(id, text string) error,
) (string, error) {
	// fetch feature with all subtasks
	full, err := fetchTask(feature.ID)
	if err != nil {
		return "", err
	}
	if len(full.Subtasks) == 0 {
		return "", nil // no children → cannot advance
	}

	featureRank := order.Rank(full.Status.Status)
	minChildRank := len(order.names) + 1 // start above max

	for _, child := range full.Subtasks {
		r := order.Rank(child.Status.Status)
		if r < minChildRank {
			minChildRank = r
		}
	}

	if minChildRank <= featureRank {
		return "", nil // children haven't moved past the feature yet
	}

	newStatus := order.Name(minChildRank)
	if newStatus == "" {
		return "", fmt.Errorf("unknown status rank %d", minChildRank)
	}

	if err := updateStatus(full.ID, newStatus); err != nil {
		return "", err
	}

	// build child summary for comment
	var childSummaries []string
	for _, child := range full.Subtasks {
		childSummaries = append(childSummaries, fmt.Sprintf("%s (%s)", child.ID, child.Status.Status))
	}
	comment := fmt.Sprintf(
		"[kha:auto] parent advanced to %s — reflects minimum status among %d children (%s).",
		newStatus,
		len(full.Subtasks),
		strings.Join(childSummaries, ", "),
	)
	if err := addComment(full.ID, comment); err != nil {
		return "", err
	}

	return newStatus, nil
}
