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

func (c *Controller) pushWinget(ctx context.Context, logger *slog.Logger, winget config.Winget, projectName, tempDir, artifactName string) error {
	// Get configuration
	forkOwner := winget.Repository.Owner
	forkName := winget.Repository.Name

	baseOwner := winget.Repository.PullRequest.Base.Owner
	baseName := winget.Repository.PullRequest.Base.Name
	if baseName == "" {
		baseName = forkName
	}
	baseBranch := winget.Repository.PullRequest.Base.Branch
	if baseBranch == "" {
		baseBranch = "master" // TODO Get the default branch by GitHub API
	}

	// Expand headBranch template: "aqua-{{.Version}}" -> "aqua-v2.0.0"
	headBranch := winget.Repository.Branch
	headBranch = strings.ReplaceAll(headBranch, "{{.Version}}", c.param.Version)
	headBranch = strings.ReplaceAll(headBranch, "{{ .Version }}", c.param.Version)
	if headBranch == "" {
		headBranch = "main" // TODO Get the default branch by GitHub API
	}

	wingetName := winget.Publisher + "." + projectName

	baseURL := fmt.Sprintf("https://github.com/%s/%s", baseOwner, baseName)
	forkURL := fmt.Sprintf("https://github.com/%s/%s", forkOwner, forkName)

	logger.Info("setting up winget repository",
		"base", baseURL,
		"fork", forkURL,
		"branch", headBranch)

	// Initialize git repository
	repoDir := filepath.Join(tempDir, "winget-pkgs")
	if err := c.exec.Run(ctx, logger, tempDir, "git", "init", "winget-pkgs"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Add base remote and fetch
	if err := c.exec.Run(ctx, logger, repoDir, "git", "remote", "add", "origin", baseURL); err != nil {
		return fmt.Errorf("add origin remote: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "fetch", "--depth=1", "origin", baseBranch); err != nil {
		return fmt.Errorf("fetch origin: %w", err)
	}

	// Create branch from origin/master
	if err := c.exec.Run(ctx, logger, repoDir, "git", "checkout", "-b", headBranch, "origin/"+baseBranch); err != nil {
		return fmt.Errorf("checkout branch: %w", err)
	}

	// Remove existing manifests directory and copy new one
	manifestsDir := filepath.Join(repoDir, "manifests")
	if err := c.fs.RemoveAll(manifestsDir); err != nil {
		return fmt.Errorf("remove manifests directory: %w", err)
	}

	srcManifestsDir := filepath.Join(tempDir, artifactName, "winget", "manifests")
	if err := c.copyDir(srcManifestsDir, manifestsDir); err != nil {
		return fmt.Errorf("copy manifests: %w", err)
	}

	// Add all manifest files
	logger.Info("committing winget changes")
	if err := c.addManifestFiles(ctx, logger, c.exec, repoDir); err != nil {
		return fmt.Errorf("add manifest files: %w", err)
	}

	commitMsg := fmt.Sprintf("Update %s to %s", wingetName, c.param.Version)
	if err := c.exec.Run(ctx, logger, repoDir, "git", "commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Add fork remote and push
	if err := c.exec.Run(ctx, logger, repoDir, "git", "remote", "add", "fork", forkURL); err != nil {
		return fmt.Errorf("add fork remote: %w", err)
	}

	if err := c.exec.Run(ctx, logger, repoDir, "git", "push", "fork", headBranch); err != nil {
		return fmt.Errorf("push to fork: %w", err)
	}

	// Create pull request
	logger.Info("creating pull request")
	if err := c.exec.Run(ctx, logger, repoDir, "gh", "repo", "set-default", baseURL); err != nil {
		return fmt.Errorf("set default repo: %w", err)
	}

	prTitle := fmt.Sprintf("New version: %s %s", wingetName, c.param.Version)
	prBody := filepath.Join(repoDir, ".github", "PULL_REQUEST_TEMPLATE.md")
	head := fmt.Sprintf("%s:%s", forkOwner, headBranch)

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
			return err
		}
		if d.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(repoDir, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	}); err != nil {
		return err
	}
	return exec.Run(ctx, logger, repoDir, "git", append([]string{"add"}, files...)...)
}

func (c *Controller) copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return c.fs.MkdirAll(dstPath, 0o755)
		}

		return c.copyFile(path, dstPath)
	})
}
