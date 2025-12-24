package run

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/suzuki-shunsuke/rgo/pkg/config"
)

type ParamRun struct {
	ConfigFilePath string
	PWD            string
	Stderr         io.Writer
	Version        string
	RunID          string
	Workflow       string
}

func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Read(c.fs, c.param.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("read a config file: %w", err)
	}

	repo, err := c.exec.Output(ctx, logger, c.param.PWD, "gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return fmt.Errorf("get repository name: %w", err)
	}

	runID := c.param.RunID

	// Create and push tag if run_id is not provided
	if runID == "" {
		if err := c.exec.Run(ctx, logger, c.param.PWD, "git", "tag", "-m", "chore: release "+c.param.Version, c.param.Version); err != nil {
			return fmt.Errorf("create a git tag: %w", err)
		}
		if err := c.exec.Run(ctx, logger, c.param.PWD, "git", "push", "origin", c.param.Version); err != nil {
			return fmt.Errorf("push a git tag: %w", err)
		}
	}

	// Skip for prerelease versions
	if strings.Contains(c.param.Version, "-") {
		logger.Info("prerelease version detected, skipping package manager updates")
		return nil
	}

	workflow := c.param.Workflow
	if workflow == "" {
		workflow = "release.yaml"
	}

	// Get run ID if not provided
	if runID == "" {
		logger.Info("waiting for workflow to start")
		time.Sleep(10 * time.Second)
		runID, err = c.exec.Output(ctx, logger, c.param.PWD, "gh", "run", "list", "-w", workflow, "-L", "1", "--json", "databaseId", "--jq", ".[].databaseId")
		if err != nil {
			return fmt.Errorf("get workflow run ID: %w", err)
		}
	}

	// Wait for workflow to complete
	logger.Info("waiting for workflow to complete", slog.String("run_id", runID))
	if err := c.exec.Run(ctx, logger, c.param.PWD, "gh", "run", "watch", "--exit-status", runID); err != nil {
		return fmt.Errorf("wait for workflow run: %w", err)
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "rgo-*")
	if err != nil {
		return fmt.Errorf("create a temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	logger.Info("created temporary directory", slog.String("path", tempDir))

	// Download artifacts
	artifactName := "goreleaser"
	logger.Info("downloading artifacts")
	if err := c.exec.Run(ctx, logger, tempDir, "gh", "run", "download", "-R", repo, runID, "--pattern", artifactName); err != nil {
		return fmt.Errorf("download artifacts: %w", err)
	}

	serverURL := os.Getenv("GITHUB_SERVER_URL")
	if serverURL == "" {
		serverURL = "https://github.com"
	}

	// Process Homebrew
	if err := c.processHomebrew(ctx, logger, cfg, tempDir, artifactName, serverURL); err != nil {
		return fmt.Errorf("process Homebrew: %w", err)
	}

	// Process Scoop
	if err := c.processScoop(ctx, logger, cfg, tempDir, artifactName, serverURL); err != nil {
		return fmt.Errorf("process Scoop: %w", err)
	}

	// Process Winget
	if err := c.processWinget(ctx, logger, cfg, tempDir, artifactName); err != nil {
		return fmt.Errorf("process Winget: %w", err)
	}

	logger.Info("release completed successfully")
	return nil
}
