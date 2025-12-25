package cmdexec

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

const waitDelay = 1000 * time.Hour

func SetCancel(cmd *exec.Cmd) {
	cmd.Cancel = func() error {
		return cmd.Process.Signal(os.Interrupt)
	}
	cmd.WaitDelay = waitDelay
}

type Executor struct {
	Stdout io.Writer
	Stderr io.Writer
}

func (e *Executor) Run(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error {
	logger.Info("executing command", "command", name, "args", args, "dir", dir)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr
	SetCancel(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute a command: %w", err)
	}
	return nil
}

func (e *Executor) Output(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) (string, error) {
	logger.Info("executing command", "command", name, "args", args, "dir", dir)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stderr = e.Stderr
	SetCancel(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("execute a command: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
