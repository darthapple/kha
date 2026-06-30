package clickup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const baseURL = "https://api.clickup.com/api/v2"

type Client struct {
	apiKey string
	http   *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey, http: &http.Client{}}
}

func (c *Client) do(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("clickup API %s %s: %d %s", method, path, resp.StatusCode, string(data))
	}
	return data, nil
}

// ListTasks fetches tasks in a given status, including subtasks.
func (c *Client) ListTasks(listID, status string) ([]TaskWithOrder, error) {
	encoded := url.QueryEscape(status)
	path := fmt.Sprintf("/list/%s/task?statuses[]=%s&subtasks=true&include_closed=true", listID, encoded)
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Tasks []TaskWithOrder `json:"tasks"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

// GetTask fetches a single task with all subtasks.
func (c *Client) GetTask(taskID string) (*TaskWithOrder, error) {
	path := fmt.Sprintf("/task/%s?include_subtasks=true", taskID)
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var t TaskWithOrder
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// GetComments fetches all comments for a task.
func (c *Client) GetComments(taskID string) ([]Comment, error) {
	path := fmt.Sprintf("/task/%s/comment", taskID)
	data, err := c.do("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp CommentListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Comments, nil
}

// UpdateTask patches task fields. Only non-nil map values are sent.
func (c *Client) UpdateTask(taskID string, fields map[string]any) error {
	_, err := c.do("PUT", "/task/"+taskID, fields)
	return err
}

// AddComment posts a comment to a task.
func (c *Client) AddComment(taskID, text string) error {
	_, err := c.do("POST", fmt.Sprintf("/task/%s/comment", taskID), map[string]any{
		"comment_text": text,
	})
	return err
}

// GetCurrentUser returns the authenticated user.
func (c *Client) GetCurrentUser() (*Member, error) {
	data, err := c.do("GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	var resp UserResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp.User, nil
}

// GetTeams returns all workspaces/teams for the authenticated user.
func (c *Client) GetTeams() ([]Team, error) {
	data, err := c.do("GET", "/team", nil)
	if err != nil {
		return nil, err
	}
	var resp TeamListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return resp.Teams, nil
}

// StartTimer starts time tracking on a task.
func (c *Client) StartTimer(teamID, taskID string) error {
	_, err := c.do("POST", fmt.Sprintf("/team/%s/time_entries/start", teamID), map[string]any{
		"tid": taskID,
	})
	return err
}

// StopTimer stops the currently running timer for the workspace.
func (c *Client) StopTimer(teamID string) error {
	_, err := c.do("POST", fmt.Sprintf("/team/%s/time_entries/stop", teamID), nil)
	return err
}

// AssignUser adds a user to a task's assignees if not already present.
func (c *Client) AssignUser(taskID string, userID int, existing []Member) error {
	for _, m := range existing {
		if m.ID == userID {
			return nil
		}
	}
	ids := make([]int, 0, len(existing)+1)
	for _, m := range existing {
		ids = append(ids, m.ID)
	}
	ids = append(ids, userID)
	return c.UpdateTask(taskID, map[string]any{"assignees": ids})
}

// UploadAttachment uploads a file to a task.
func (c *Client) UploadAttachment(taskID, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("attachment", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return err
	}
	w.Close()

	req, err := http.NewRequest("POST", baseURL+fmt.Sprintf("/task/%s/attachment", taskID), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload attachment: %d %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetList fetches a ClickUp list (including its current statuses).
func (c *Client) GetList(listID string) (*List, error) {
	data, err := c.do("GET", "/list/"+listID, nil)
	if err != nil {
		return nil, err
	}
	var l List
	if err := json.Unmarshal(data, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// EnsureListStatus creates a new status on a list if it does not already exist.
// Returns (true, nil) when created, (false, nil) when it already existed.
// before/after control insertion position by matching an existing status name (case-insensitive).
func (c *Client) EnsureListStatus(listID, name, color, before, after string) (bool, error) {
	list, err := c.GetList(listID)
	if err != nil {
		return false, err
	}

	for _, s := range list.Statuses {
		if strings.EqualFold(s.Status, name) {
			return false, nil
		}
	}

	newS := StatusPut{Name: name, Color: color, Type: "custom"}
	puts := make([]StatusPut, 0, len(list.Statuses)+1)
	inserted := false

	for _, s := range list.Statuses {
		if !inserted && before != "" && strings.EqualFold(s.Status, before) {
			puts = append(puts, newS)
			inserted = true
		}
		puts = append(puts, StatusPut{ID: s.ID, Name: s.Status, Color: s.Color, Type: s.Type})
		if !inserted && after != "" && strings.EqualFold(s.Status, after) {
			puts = append(puts, newS)
			inserted = true
		}
	}
	if !inserted {
		puts = append(puts, newS)
	}

	if _, err := c.do("PUT", "/list/"+listID, map[string]any{"statuses": puts}); err != nil {
		return false, err
	}
	return true, nil
}

// CreateTask creates a new task in the given list.
func (c *Client) CreateTask(listID, name, status, taskType, parentID, description string) (*Task, error) {
	body := map[string]any{
		"name":   name,
		"status": status,
	}
	if taskType != "" {
		body["task_type"] = taskType
	}
	if parentID != "" {
		body["parent"] = parentID
	}
	if description != "" {
		body["description"] = description
	}
	data, err := c.do("POST", "/list/"+listID+"/task", body)
	if err != nil {
		return nil, err
	}
	var t Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// ResolveNewlines converts literal \n sequences in flag values to real newlines.
func ResolveNewlines(s string) string {
	return strings.ReplaceAll(s, `\n`, "\n")
}
