package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/config"
)

func (c *Controller) processScoop(ctx context.Context, logger *slog.Logger, cfg *config.Config, tempDir, artifactName, serverURL string) error {
	scoopDir := filepath.Join(tempDir, artifactName, "scoop")
	if _, err := c.fs.Stat(scoopDir); os.IsNotExist(err) {
		logger.Info("Scoop manifest isn't found")
		return nil
	}

	for _, scoop := range cfg.Scoops {
		if err := c.pushScoop(ctx, logger, scoop.Repository, cfg.ProjectName, tempDir, artifactName, serverURL); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) pushScoop(ctx context.Context, logger *slog.Logger, repo config.Repository, projectName, tempDir, artifactName, serverURL string) error {
	repoURL := fmt.Sprintf("%s/%s/%s", serverURL, repo.Owner, repo.Name)

	logger.Info("cloning scoop repository", "repo", repoURL)
	repoDir := filepath.Join(tempDir, repo.Name)
	if err := c.exec.Run(ctx, logger, tempDir, "git", "clone", "--depth", "1", repoURL); err != nil {
		return fmt.Errorf("clone scoop repository: %w", err)
	}

	if err := c.copyScoopFiles(tempDir, artifactName, repoDir); err != nil {
		return err
	}

	logger.Info("committing and pushing scoop changes")
	if err := c.commitAndPushScoop(ctx, logger, repo, projectName, repoDir); err != nil {
		return err
	}

	return nil
}

func (c *Controller) copyScoopFiles(tempDir, artifactName, repoDir string) error {
	scoopDir := filepath.Join(tempDir, artifactName, "scoop")
	entries, err := afero.ReadDir(c.fs, scoopDir)
	if err != nil {
		return fmt.Errorf("read scoop directory: %w", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		src := filepath.Join(scoopDir, entry.Name())
		dst := filepath.Join(repoDir, entry.Name())
		if err := c.copyFile(src, dst); err != nil {
			return fmt.Errorf("copy scoop file: %w", err)
		}
	}
	return nil
}

func (c *Controller) commitAndPushScoop(ctx context.Context, logger *slog.Logger, repo config.Repository, projectName, repoDir string) error {
	if err := c.exec.Run(ctx, logger, repoDir, "git", "add", "*.json"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	commitMsg := fmt.Sprintf("Scoop update for %s version %s", projectName, c.param.Version)
	if err := c.exec.Run(ctx, logger, repoDir, "git", "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	branch, err := c.getBranch(ctx, logger, repo)
	if err != nil {
		return err
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "push", "origin", branch); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

func (c *Controller) getBranch(ctx context.Context, logger *slog.Logger, repo config.Repository) (string, error) {
	if repo.Branch != "" {
		return repo.Branch, nil
	}
	branch, err := c.getDefaultBranch(ctx, logger, repo.Owner, repo.Name)
	if err != nil {
		return "", fmt.Errorf("get default branch: %w", err)
	}
	return branch, nil
}
