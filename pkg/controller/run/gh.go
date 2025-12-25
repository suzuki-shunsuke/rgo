package run

import (
	"context"
	"fmt"
	"log/slog"
)

func (c *Controller) getDefaultBranch(ctx context.Context, logger *slog.Logger, owner, repo string) (string, error) {
	logger.Info("getting default branch", "owner", owner, "repo", repo)
	r, _, err := c.ghRepo.Get(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("get repository: %w", err)
	}
	return r.GetDefaultBranch(), nil
}

func (c *Controller) getRunID(ctx context.Context, logger *slog.Logger, workflow string) (string, error) {
	runID, err := c.exec.Output(ctx, logger, "", "gh", "run", "list", "-w", workflow, "-L", "1", "--json", "databaseId", "--jq", ".[].databaseId")
	if err != nil {
		return "", fmt.Errorf("get workflow run ID: %w", err)
	}
	return runID, nil
}

func (c *Controller) watchRun(ctx context.Context, logger *slog.Logger, runID string) error {
	if err := c.exec.Run(ctx, logger, "", "gh", "run", "watch", "--exit-status", runID); err != nil {
		return fmt.Errorf("wait for workflow run: %w", err)
	}
	return nil
}

func (c *Controller) downloadArtifacts(ctx context.Context, logger *slog.Logger, dir, repo, runID string) error {
	if err := c.exec.Run(ctx, logger, dir, "gh", "run", "download", "-R", repo, runID, "--pattern", "goreleaser"); err != nil {
		return fmt.Errorf("download artifacts: %w", err)
	}
	return nil
}
