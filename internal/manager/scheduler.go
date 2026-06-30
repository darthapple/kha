package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/darthapple/kha/internal/executor"
	"github.com/darthapple/kha/internal/slots"
)

// Scheduler polls ClickUp for pending tasks and dispatches skill containers,
// enforcing one active container per pipeline step.
type Scheduler struct {
	cfg      *Config
	slots    *slots.Store
	executor executor.Executor
}

func NewScheduler(cfg *Config, store *slots.Store, exec executor.Executor) *Scheduler {
	return &Scheduler{cfg: cfg, slots: store, executor: exec}
}

// Run blocks until ctx is cancelled, polling ClickUp on each tick.
func (s *Scheduler) Run(ctx context.Context) {
	log.Printf("manager: poll_interval=%s skill_image=%s", s.cfg.PollInterval, s.cfg.SkillImage)

	s.poll(ctx)

	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *Scheduler) poll(ctx context.Context) {
	for _, step := range s.cfg.Steps {
		if step.Skill == "" {
			continue
		}

		occupied, err := s.slots.IsOccupied(ctx, step.Skill)
		if err != nil {
			log.Printf("[%s] slot check failed: %v", step.Skill, err)
			continue
		}
		if occupied {
			log.Printf("[%s] slot occupied — skipping", step.Skill)
			continue
		}

		taskID, taskName, err := s.firstTask(ctx, step.Status)
		if err != nil {
			log.Printf("[%s] task query failed: %v", step.Skill, err)
			continue
		}
		if taskID == "" {
			continue // no work
		}

		if err := s.slots.Acquire(ctx, step.Skill, taskID); err != nil {
			log.Printf("[%s] acquire slot failed: %v", step.Skill, err)
			continue
		}

		log.Printf("[%s] dispatching task %s: %s", step.Skill, taskID, taskName)
		go s.run(ctx, step.Skill, taskID, taskName)
	}
}

func (s *Scheduler) run(ctx context.Context, skill, taskID, taskName string) {
	defer func() {
		if err := s.slots.Release(ctx, skill); err != nil {
			log.Printf("[%s] slot release failed: %v", skill, err)
		}
	}()

	result, err := s.executor.Run(ctx, skill)
	if err != nil {
		log.Printf("[%s] task %s executor error: %v", skill, taskID, err)
		return
	}

	if result.ExitCode != 0 {
		log.Printf("[%s] task %s (%s) failed (exit=%d):\n%s",
			skill, taskID, taskName, result.ExitCode, result.Logs)
		return
	}

	log.Printf("[%s] task %s (%s) done", skill, taskID, taskName)
}

// firstTask calls `kha next` and returns the first task's ID and name,
// or empty strings when there is no work.
func (s *Scheduler) firstTask(ctx context.Context, status string) (id, name string, err error) {
	cmd := exec.CommandContext(ctx,
		"/root/.kha/kha", "next", status,
		"--list", s.cfg.ClickUpListID,
		"--pipeline", s.cfg.Pipeline,
	)

	out, execErr := cmd.Output()
	if execErr != nil {
		return "", "", fmt.Errorf("kha next: %w", execErr)
	}

	var result struct {
		Tasks []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return "", "", fmt.Errorf("parse output: %w", err)
	}

	if len(result.Tasks) == 0 {
		return "", "", nil
	}
	return result.Tasks[0].ID, result.Tasks[0].Name, nil
}
