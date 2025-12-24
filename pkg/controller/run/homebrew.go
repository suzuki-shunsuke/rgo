package run

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/suzuki-shunsuke/rgo/pkg/config"
)

func (c *Controller) processHomebrew(ctx context.Context, logger *slog.Logger, cfg *config.Config, tempDir, artifactName, serverURL string) error {
	homebrewDir := filepath.Join(tempDir, artifactName, "homebrew")
	if _, err := os.Stat(homebrewDir); os.IsNotExist(err) {
		logger.Info("Homebrew-tap recipe isn't found")
		return nil
	}

	// Process homebrew_casks
	for _, cask := range cfg.HomebrewCasks {
		if err := c.pushHomebrew(ctx, logger, cask.Repository, cfg.ProjectName, tempDir, artifactName, serverURL); err != nil {
			return err
		}
	}

	// Process brews (traditional formula)
	for _, brew := range cfg.Brews {
		if err := c.pushHomebrew(ctx, logger, brew.Repository, cfg.ProjectName, tempDir, artifactName, serverURL); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) pushHomebrew(ctx context.Context, logger *slog.Logger, repo config.Repository, projectName, tempDir, artifactName, serverURL string) error {
	repoName := "homebrew-" + repo.Name
	if repo.Name == "" {
		repoName = "homebrew-" + projectName
	}
	repoURL := fmt.Sprintf("%s/%s/%s", serverURL, repo.Owner, repoName)

	logger.Info("cloning homebrew repository", "repo", repoURL)
	repoDir := filepath.Join(tempDir, repoName)
	if err := c.exec.Run(ctx, logger, tempDir, "git", "clone", "--depth", "1", repoURL); err != nil {
		return fmt.Errorf("clone homebrew repository: %w", err)
	}

	// Copy homebrew files
	homebrewDir := filepath.Join(tempDir, artifactName, "homebrew")
	entries, err := os.ReadDir(homebrewDir)
	if err != nil {
		return fmt.Errorf("read homebrew directory: %w", err)
	}

	for _, entry := range entries {
		src := filepath.Join(homebrewDir, entry.Name())
		dst := filepath.Join(repoDir, entry.Name())
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy homebrew file: %w", err)
		}
	}

	// Commit and push
	logger.Info("committing and pushing homebrew changes")
	if err := c.exec.Run(ctx, logger, repoDir, "git", "add", "."); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	commitMsg := fmt.Sprintf("Brew formula update for %s version %s", projectName, c.param.Version)
	if err := c.exec.Run(ctx, logger, repoDir, "git", "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "push", "origin", "main"); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644) //nolint:gosec
}
