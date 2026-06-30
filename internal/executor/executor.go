package executor

import "context"

type RunResult struct {
	ExitCode int
	Logs     string
}

type Executor interface {
	Run(ctx context.Context, skill string) (RunResult, error)
}
