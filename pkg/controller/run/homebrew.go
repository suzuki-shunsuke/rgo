package run

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/config"
)

func (c *Controller) processHomebrew(ctx context.Context, logger *slog.Logger, cfg *config.Config, tempDir, artifactName, serverURL string) error {
	homebrewDir := filepath.Join(tempDir, artifactName, "homebrew")
	if exists, err := afero.Exists(c.fs, homebrewDir); err != nil {
		return fmt.Errorf("check homebrew directory existence: %w", err)
	} else if !exists {
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
	repoURL := fmt.Sprintf("%s/%s/%s", serverURL, repo.Owner, repo.Name)

	logger.Info("cloning homebrew repository", "repo", repoURL)
	repoDir := filepath.Join(tempDir, repo.Name)
	if err := c.exec.Run(ctx, logger, tempDir, "git", "clone", "--depth", "1", repoURL); err != nil {
		return fmt.Errorf("clone homebrew repository: %w", err)
	}

	// Copy homebrew files
	homebrewDir := filepath.Join(tempDir, artifactName, "homebrew")
	if err := c.copyDir(homebrewDir, repoDir); err != nil {
		return fmt.Errorf("copy homebrew directory: %w", err)
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

	branch := repo.Branch
	if branch == "" {
		var err error
		branch, err = c.getDefaultBranch(ctx, logger, repo.Owner, repo.Name)
		if err != nil {
			return fmt.Errorf("get default branch: %w", err)
		}
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "push", "origin", branch); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

const filePermission = 0o644

func (c *Controller) copyFile(src, dst string) error {
	data, err := afero.ReadFile(c.fs, src)
	if err != nil {
		return fmt.Errorf("read file %s: %w", src, err)
	}
	if err := afero.WriteFile(c.fs, dst, data, filePermission); err != nil {
		return fmt.Errorf("write file %s: %w", dst, err)
	}
	return nil
}
