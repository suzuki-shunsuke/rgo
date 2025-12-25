package run

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/config"
)

func (c *Controller) processWinget(ctx context.Context, logger *slog.Logger, cfg *config.Config, tempDir, artifactName string) error {
	wingetDir := filepath.Join(tempDir, artifactName, "winget")
	if _, err := c.fs.Stat(wingetDir); os.IsNotExist(err) {
		logger.Info("Winget manifest isn't found")
		return nil
	}

	for _, winget := range cfg.Winget {
		if err := c.pushWinget(ctx, logger, winget, cfg.ProjectName, tempDir, artifactName); err != nil {
			return err
		}
	}

	return nil
}

type wingetConfig struct {
	forkOwner  string
	forkName   string
	baseOwner  string
	baseName   string
	baseBranch string
	headBranch string
	baseURL    string
	forkURL    string
	wingetName string
}

func (c *Controller) pushWinget(ctx context.Context, logger *slog.Logger, winget config.Winget, projectName, tempDir, artifactName string) error {
	cfg, err := c.buildWingetConfig(ctx, logger, winget, projectName)
	if err != nil {
		return err
	}

	repoDir, err := c.setupWingetRepo(ctx, logger, tempDir, cfg)
	if err != nil {
		return err
	}

	if err := c.updateWingetManifests(ctx, logger, tempDir, artifactName, repoDir, cfg.wingetName); err != nil {
		return err
	}

	if err := c.pushWingetToFork(ctx, logger, repoDir, cfg); err != nil {
		return err
	}

	return c.createWingetPR(ctx, logger, repoDir, cfg)
}

func (c *Controller) buildWingetConfig(ctx context.Context, logger *slog.Logger, winget config.Winget, projectName string) (*wingetConfig, error) {
	cfg := &wingetConfig{
		forkOwner: winget.Repository.Owner,
		forkName:  winget.Repository.Name,
		baseOwner: winget.Repository.PullRequest.Base.Owner,
		baseName:  winget.Repository.PullRequest.Base.Name,
	}

	if cfg.baseName == "" {
		cfg.baseName = cfg.forkName
	}

	cfg.baseBranch = winget.Repository.PullRequest.Base.Branch
	if cfg.baseBranch == "" {
		var err error
		cfg.baseBranch, err = c.getDefaultBranch(ctx, logger, cfg.baseOwner, cfg.baseName)
		if err != nil {
			return nil, fmt.Errorf("get base repository default branch: %w", err)
		}
	}

	cfg.headBranch = winget.Repository.Branch
	cfg.headBranch = strings.ReplaceAll(cfg.headBranch, "{{.Version}}", c.param.Version)
	cfg.headBranch = strings.ReplaceAll(cfg.headBranch, "{{ .Version }}", c.param.Version)
	if cfg.headBranch == "" {
		var err error
		cfg.headBranch, err = c.getDefaultBranch(ctx, logger, cfg.forkOwner, cfg.forkName)
		if err != nil {
			return nil, fmt.Errorf("get fork repository default branch: %w", err)
		}
	}

	cfg.wingetName = winget.Publisher + "." + projectName
	cfg.baseURL = fmt.Sprintf("https://github.com/%s/%s", cfg.baseOwner, cfg.baseName)
	cfg.forkURL = fmt.Sprintf("https://github.com/%s/%s", cfg.forkOwner, cfg.forkName)

	return cfg, nil
}

func (c *Controller) setupWingetRepo(ctx context.Context, logger *slog.Logger, tempDir string, cfg *wingetConfig) (string, error) {
	logger.Info("setting up winget repository",
		"base", cfg.baseURL,
		"fork", cfg.forkURL,
		"branch", cfg.headBranch)

	repoDir := filepath.Join(tempDir, "winget-pkgs")
	if err := c.exec.Run(ctx, logger, tempDir, "git", "init", "winget-pkgs"); err != nil {
		return "", fmt.Errorf("git init: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "remote", "add", "origin", cfg.baseURL); err != nil {
		return "", fmt.Errorf("add origin remote: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "fetch", "--depth=1", "origin", cfg.baseBranch); err != nil {
		return "", fmt.Errorf("fetch origin: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "checkout", "-b", cfg.headBranch, "origin/"+cfg.baseBranch); err != nil {
		return "", fmt.Errorf("checkout branch: %w", err)
	}

	return repoDir, nil
}

func (c *Controller) updateWingetManifests(ctx context.Context, logger *slog.Logger, tempDir, artifactName, repoDir, wingetName string) error {
	manifestsDir := filepath.Join(repoDir, "manifests")
	if err := c.fs.RemoveAll(manifestsDir); err != nil {
		return fmt.Errorf("remove manifests directory: %w", err)
	}

	srcManifestsDir := filepath.Join(tempDir, artifactName, "winget", "manifests")
	if err := c.copyDir(srcManifestsDir, manifestsDir); err != nil {
		return fmt.Errorf("copy manifests: %w", err)
	}

	logger.Info("committing winget changes")
	if err := c.addManifestFiles(ctx, logger, c.exec, repoDir); err != nil {
		return fmt.Errorf("add manifest files: %w", err)
	}

	commitMsg := fmt.Sprintf("Update %s to %s", wingetName, c.param.Version)
	if err := c.exec.Run(ctx, logger, repoDir, "git", "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	return nil
}

func (c *Controller) pushWingetToFork(ctx context.Context, logger *slog.Logger, repoDir string, cfg *wingetConfig) error {
	if err := c.exec.Run(ctx, logger, repoDir, "git", "remote", "add", "fork", cfg.forkURL); err != nil {
		return fmt.Errorf("add fork remote: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "push", "fork", cfg.headBranch); err != nil {
		return fmt.Errorf("push to fork: %w", err)
	}

	return nil
}

func (c *Controller) createWingetPR(ctx context.Context, logger *slog.Logger, repoDir string, cfg *wingetConfig) error {
	logger.Info("creating pull request")
	if err := c.exec.Run(ctx, logger, repoDir, "gh", "repo", "set-default", cfg.baseURL); err != nil {
		return fmt.Errorf("set default repo: %w", err)
	}

	prTitle := fmt.Sprintf("New version: %s %s", cfg.wingetName, c.param.Version)
	prBody := filepath.Join(repoDir, ".github", "PULL_REQUEST_TEMPLATE.md")
	head := fmt.Sprintf("%s:%s", cfg.forkOwner, cfg.headBranch)

	prArgs := []string{"pr", "create", "--title", prTitle, "--head", head}
	if _, err := c.fs.Stat(prBody); err == nil {
		prArgs = append(prArgs, "--body-file", prBody)
	} else {
		prArgs = append(prArgs, "--body", "")
	}
	prArgs = append(prArgs, "--web")

	if err := c.exec.Run(ctx, logger, repoDir, "gh", prArgs...); err != nil {
		return fmt.Errorf("create pull request: %w", err)
	}

	return nil
}

func (c *Controller) addManifestFiles(ctx context.Context, logger *slog.Logger, exec interface {
	Run(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error
}, repoDir string,
) error {
	manifestsDir := filepath.Join(repoDir, "manifests")
	files := []string{}
	if err := fs.WalkDir(afero.NewIOFS(c.fs), manifestsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk directory: %w", err)
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(repoDir, path)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}
		files = append(files, relPath)
		return nil
	}); err != nil {
		return fmt.Errorf("walk manifests directory: %w", err)
	}
	if err := exec.Run(ctx, logger, repoDir, "git", append([]string{"add"}, files...)...); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	return nil
}

func (c *Controller) copyDir(src, dst string) error {
	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk directory: %w", err)
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("get relative path: %w", err)
		}
		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			if err := c.fs.MkdirAll(dstPath, 0o755); err != nil { //nolint:mnd
				return fmt.Errorf("create directory %s: %w", dstPath, err)
			}
			return nil
		}

		return c.copyFile(path, dstPath)
	}); err != nil {
		return fmt.Errorf("copy directory: %w", err)
	}
	return nil
}
