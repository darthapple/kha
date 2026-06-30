package clickup

import (
	"encoding/json"
	"strconv"
	"strings"
)

type TaskListResponse struct {
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Assignees   []Member   `json:"assignees"`
	Parent      *string    `json:"parent"`
	URL         string     `json:"url"`
	TeamID      string     `json:"team_id"`
	Subtasks    []Task     `json:"subtasks"`
	RawType     rawType    `json:"task_type"`
}

// rawType handles ClickUp returning task_type as string, int, or null.
type rawType struct {
	value string
}

func (r *rawType) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}
	// try string
	var s string
	if json.Unmarshal(data, &s) == nil {
		r.value = s
		return nil
	}
	// try number → convert to string
	var n json.Number
	if json.Unmarshal(data, &n) == nil {
		r.value = n.String()
		return nil
	}
	return nil
}

func (r rawType) String() string { return r.value }

// OrderIndex is returned as a string decimal by ClickUp.
type OrderIndex struct {
	f float64
}

func (o *OrderIndex) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	o.f = f
	return nil
}

func (o OrderIndex) F() float64 { return o.f }

// TaskWithOrder wraps Task with its orderindex (returned as string by ClickUp).
type TaskWithOrder struct {
	Task
	OrderIndex OrderIndex `json:"orderindex"`
}

type Status struct {
	ID         string `json:"id,omitempty"`
	Status     string `json:"status"`
	Color      string `json:"color"`
	OrderIndex int    `json:"orderindex"`
	Type       string `json:"type"`
}

// List holds the fields returned by GET /list/{id} that we care about.
type List struct {
	ID       string   `json:"id"`
	Statuses []Status `json:"statuses"`
}

// StatusPut is the shape sent inside PUT /list/{id} statuses array.
// Using a separate type avoids sending orderindex, which ClickUp derives from array position.
type StatusPut struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"status"`
	Color string `json:"color"`
	Type  string `json:"type"`
}

type Member struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type Comment struct {
	ID   string `json:"id"`
	Text string `json:"comment_text"`
	Date string `json:"date"`
	User Member `json:"user"`
}

type CommentListResponse struct {
	Comments []Comment `json:"comments"`
}

type UserResponse struct {
	User Member `json:"user"`
}

type TeamListResponse struct {
	Teams []Team `json:"teams"`
}

type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type TimeEntry struct {
	ID   string `json:"id"`
	Task struct {
		ID string `json:"id"`
	} `json:"task"`
}

type TimeEntryResponse struct {
	Data TimeEntry `json:"data"`
}
