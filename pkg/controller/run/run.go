package run

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/config"
)

type ParamRun struct {
	ConfigFilePath string
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

	runID := c.param.RunID

	// Create and push tag if run_id is not provided
	if runID == "" {
		if err := c.createTag(ctx, logger, c.param.Version); err != nil {
			return err
		}
		if err := c.pushTag(ctx, logger, c.param.Version); err != nil {
			return err
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
		runID, err = c.getRunID(ctx, logger, workflow)
		if err != nil {
			return err
		}
	}

	// Wait for workflow to complete
	logger.Info("waiting for workflow to complete", "run_id", runID)
	if err := c.watchRun(ctx, logger, runID); err != nil {
		return err
	}

	// Create temporary directory
	tempDir, err := afero.TempDir(c.fs, "", "rgo-")
	if err != nil {
		return fmt.Errorf("create a temporary directory: %w", err)
	}
	logger.Info("created temporary directory", "path", tempDir)

	// Download artifacts
	artifactName := "goreleaser"
	logger.Info("downloading artifacts")
	if err := c.downloadArtifacts(ctx, logger, tempDir, runID); err != nil {
		return err
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
