package executor

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// DockerConfig holds everything needed to spawn a skill container.
type DockerConfig struct {
	Image               string
	ClickUpAPIKey       string
	AnthropicAPIKey     string // API key auth (sk-ant-api03-...)
	ClaudeOAuthToken    string // OAuth token from `claude setup-token` → CLAUDE_CODE_OAUTH_TOKEN
	RepoURL             string
	GitToken            string
	SocketPath          string // default: /var/run/docker.sock
}

// DockerExecutor runs skills as ephemeral Docker containers via the Engine API.
// It uses the Unix socket directly to avoid the Docker SDK dependency.
type DockerExecutor struct {
	http http.Client
	cfg  DockerConfig
}

func NewDockerExecutor(cfg DockerConfig) *DockerExecutor {
	sock := cfg.SocketPath
	if sock == "" {
		sock = "/var/run/docker.sock"
	}
	return &DockerExecutor{
		http: http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", sock)
				},
			},
		},
		cfg: cfg,
	}
}

func (e *DockerExecutor) Run(ctx context.Context, skill string) (RunResult, error) {
	containerID, err := e.createContainer(ctx, skill)
	if err != nil {
		return RunResult{}, fmt.Errorf("create container: %w", err)
	}

	if err := e.startContainer(ctx, containerID); err != nil {
		e.removeContainer(context.Background(), containerID)
		return RunResult{}, fmt.Errorf("start container: %w", err)
	}

	exitCode, err := e.waitContainer(ctx, containerID)
	if err != nil {
		e.removeContainer(context.Background(), containerID)
		return RunResult{}, fmt.Errorf("wait container: %w", err)
	}

	logs := e.fetchLogs(ctx, containerID)
	e.removeContainer(context.Background(), containerID)

	return RunResult{ExitCode: exitCode, Logs: logs}, nil
}

func (e *DockerExecutor) createContainer(ctx context.Context, skill string) (string, error) {
	env := e.buildEnv(skill)
	body := map[string]any{
		"Image":      e.cfg.Image,
		"Env":        env,
		"Entrypoint": []string{"/usr/local/bin/docker-entrypoint-once.sh"},
		"Cmd":        []string{},
		"HostConfig": e.buildHostConfig(),
	}

	resp, err := e.doRequest(ctx, http.MethodPost, "/v1.41/containers/create", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}

	var result struct {
		ID string `json:"Id"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.ID, nil
}

func (e *DockerExecutor) startContainer(ctx context.Context, id string) error {
	resp, err := e.doRequest(ctx, http.MethodPost, "/v1.41/containers/"+id+"/start", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (e *DockerExecutor) waitContainer(ctx context.Context, id string) (int, error) {
	resp, err := e.doRequest(ctx, http.MethodPost, "/v1.41/containers/"+id+"/wait?condition=not-running", nil)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	var result struct {
		StatusCode int `json:"StatusCode"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.StatusCode, nil
}

func (e *DockerExecutor) fetchLogs(ctx context.Context, id string) string {
	resp, err := e.doRequest(ctx, http.MethodGet, "/v1.41/containers/"+id+"/logs?stdout=1&stderr=1&tail=100", nil)
	if err != nil || resp == nil {
		return ""
	}
	defer resp.Body.Close()
	return demultiplex(resp.Body)
}

func (e *DockerExecutor) removeContainer(ctx context.Context, id string) {
	resp, _ := e.doRequest(ctx, http.MethodDelete, "/v1.41/containers/"+id+"?force=true&v=true", nil)
	if resp != nil {
		resp.Body.Close()
	}
}

func (e *DockerExecutor) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var r io.Reader
	if body != nil {
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(body)
		r = &buf
	}
	req, err := http.NewRequestWithContext(ctx, method, "http://docker"+path, r)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return e.http.Do(req)
}

func (e *DockerExecutor) buildHostConfig() map[string]any {
	return nil
}

func (e *DockerExecutor) buildEnv(skill string) []string {
	env := []string{
		"KHA_SKILL=" + skill,
		"KHA_MODE=auto",
		"CLICKUP_API_KEY=" + e.cfg.ClickUpAPIKey,
		"PROJECT_REPO_URL=" + e.cfg.RepoURL,
	}
	if e.cfg.AnthropicAPIKey != "" {
		env = append(env, "ANTHROPIC_API_KEY="+e.cfg.AnthropicAPIKey)
	}
	if e.cfg.ClaudeOAuthToken != "" {
		env = append(env, "CLAUDE_CODE_OAUTH_TOKEN="+e.cfg.ClaudeOAuthToken)
	}
	if e.cfg.GitToken != "" {
		env = append(env, "GIT_TOKEN="+e.cfg.GitToken)
	}
	return env
}

// demultiplex strips Docker's 8-byte multiplexed stream headers from log output.
func demultiplex(r io.Reader) string {
	var sb strings.Builder
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			break
		}
		size := binary.BigEndian.Uint32(header[4:])
		frame := make([]byte, size)
		if _, err := io.ReadFull(r, frame); err != nil {
			break
		}
		sb.Write(frame)
	}
	return sb.String()
}
