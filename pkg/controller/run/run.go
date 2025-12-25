package run

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
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
	Publish        []string
}

func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	cfg, err := config.Read(c.fs, c.param.ConfigFilePath)
	if err != nil {
		return fmt.Errorf("read a config file: %w", err)
	}

	runID, err := c.prepareRelease(ctx, logger)
	if err != nil {
		return err
	}

	// Skip for prerelease versions
	if strings.Contains(c.param.Version, "-") {
		logger.Info("prerelease version detected, skipping package manager updates")
		return nil
	}

	runID, err = c.waitForWorkflow(ctx, logger, runID)
	if err != nil {
		return err
	}

	tempDir, err := c.downloadReleaseArtifacts(ctx, logger, runID)
	if err != nil {
		return err
	}

	if err := c.publishPackages(ctx, logger, cfg, tempDir); err != nil {
		return err
	}

	logger.Info("release completed successfully")
	return nil
}

func (c *Controller) prepareRelease(ctx context.Context, logger *slog.Logger) (string, error) {
	runID := c.param.RunID
	if runID != "" {
		return runID, nil
	}

	if err := c.createTag(ctx, logger, c.param.Version); err != nil {
		return "", err
	}
	if err := c.pushTag(ctx, logger, c.param.Version); err != nil {
		return "", err
	}
	return "", nil
}

func (c *Controller) waitForWorkflow(ctx context.Context, logger *slog.Logger, runID string) (string, error) {
	workflow := c.param.Workflow
	if workflow == "" {
		workflow = "release.yaml"
	}

	if runID == "" {
		logger.Info("waiting for workflow to start")
		if err := wait(ctx, 10*time.Second); err != nil { //nolint:mnd
			return "", err
		}
		var err error
		runID, err = c.getRunID(ctx, logger, workflow)
		if err != nil {
			return "", err
		}
	}

	logger.Info("waiting for workflow to complete", "run_id", runID)
	if err := c.watchRun(ctx, logger, runID); err != nil {
		return "", err
	}
	return runID, nil
}

func (c *Controller) downloadReleaseArtifacts(ctx context.Context, logger *slog.Logger, runID string) (string, error) {
	tempDir, err := afero.TempDir(c.fs, "", "rgo-")
	if err != nil {
		return "", fmt.Errorf("create a temporary directory: %w", err)
	}
	logger.Info("created temporary directory", "path", tempDir)

	logger.Info("downloading artifacts")
	if err := c.downloadArtifacts(ctx, logger, tempDir, runID); err != nil {
		return "", err
	}
	return tempDir, nil
}

func (c *Controller) publishPackages(ctx context.Context, logger *slog.Logger, cfg *config.Config, tempDir string) error {
	serverURL := os.Getenv("GITHUB_SERVER_URL")
	if serverURL == "" {
		serverURL = "https://github.com"
	}
	artifactName := "goreleaser"

	if c.shouldPublish("homebrew") {
		if err := c.processHomebrew(ctx, logger, cfg, tempDir, artifactName, serverURL); err != nil {
			return fmt.Errorf("process Homebrew: %w", err)
		}
	}

	if c.shouldPublish("scoop") {
		if err := c.processScoop(ctx, logger, cfg, tempDir, artifactName, serverURL); err != nil {
			return fmt.Errorf("process Scoop: %w", err)
		}
	}

	if c.shouldPublish("winget") {
		if err := c.processWinget(ctx, logger, cfg, tempDir, artifactName); err != nil {
			return fmt.Errorf("process Winget: %w", err)
		}
	}
	return nil
}

func wait(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait interrupted: %w", ctx.Err())
	}
}

func (c *Controller) shouldPublish(name string) bool {
	if len(c.param.Publish) == 0 {
		return true
	}
	return slices.Contains(c.param.Publish, name)
}
