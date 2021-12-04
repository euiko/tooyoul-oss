package miner

import (
	"context"
	"os/exec"
	"path"
)

type (
	Executor interface {
		Execute(ctx context.Context, name string, args []string) *exec.Cmd
	}
	PathExecutor struct {
		basePath string
	}
)

func (e *PathExecutor) Execute(ctx context.Context, name string, args []string) *exec.Cmd {
	if e.basePath != "" {
		return exec.CommandContext(ctx, path.Join(e.basePath, name), args...)
	}
	return exec.CommandContext(ctx, name, args...)
}

func NewPathExecutor(basePath string) *PathExecutor {
	executor := new(PathExecutor)
	executor.basePath = basePath
	return executor
}
