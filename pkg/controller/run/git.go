package run

import (
	"context"
	"fmt"
	"log/slog"
)

func (c *Controller) createTag(ctx context.Context, logger *slog.Logger, version string) error {
	if err := c.exec.Run(ctx, logger, "", "git", "tag", "-m", "chore: release "+version, version); err != nil {
		return fmt.Errorf("create a git tag: %w", err)
	}
	return nil
}

func (c *Controller) pushTag(ctx context.Context, logger *slog.Logger, version string) error {
	if err := c.exec.Run(ctx, logger, "", "git", "push", "origin", version); err != nil {
		return fmt.Errorf("push a git tag: %w", err)
	}
	return nil
}
