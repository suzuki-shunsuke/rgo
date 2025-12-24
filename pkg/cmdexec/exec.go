package cmdexec

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
)

type Executor struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (e *Executor) Run(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error {
	logger.Info("executing command", slog.String("command", name), slog.Any("args", args), slog.String("dir", dir))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute a command: %w", err)
	}
	return nil
}

func (e *Executor) Output(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) (string, error) {
	logger.Info("executing command", slog.String("command", name), slog.Any("args", args), slog.String("dir", dir))
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = e.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("execute a command: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
