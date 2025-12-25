package run

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-github/v80/github"
	"github.com/spf13/afero"
)

// Mock Executor
type mockExecutor struct {
	runFunc    func(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error
	outputFunc func(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) (string, error)
}

func (m *mockExecutor) Run(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error {
	if m.runFunc != nil {
		return m.runFunc(ctx, logger, dir, name, args...)
	}
	return nil
}

func (m *mockExecutor) Output(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) (string, error) {
	if m.outputFunc != nil {
		return m.outputFunc(ctx, logger, dir, name, args...)
	}
	return "", nil
}

// Mock RepositoriesClient
type mockRepositoriesClient struct {
	getFunc func(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
}

func (m *mockRepositoriesClient) Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, owner, repo)
	}
	return &github.Repository{}, nil, nil
}

func TestController_shouldPublish(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		publish []string
		target  string
		want    bool
	}{
		{
			name:    "empty publish list returns true",
			publish: nil,
			target:  "homebrew",
			want:    true,
		},
		{
			name:    "empty slice returns true",
			publish: []string{},
			target:  "scoop",
			want:    true,
		},
		{
			name:    "target in list returns true",
			publish: []string{"homebrew", "scoop"},
			target:  "scoop",
			want:    true,
		},
		{
			name:    "target not in list returns false",
			publish: []string{"homebrew", "scoop"},
			target:  "winget",
			want:    false,
		},
		{
			name:    "single item match",
			publish: []string{"winget"},
			target:  "winget",
			want:    true,
		},
		{
			name:    "single item no match",
			publish: []string{"winget"},
			target:  "homebrew",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Controller{
				param: &ParamRun{
					Publish: tt.publish,
				},
			}
			if got := c.shouldPublish(tt.target); got != tt.want {
				t.Errorf("shouldPublish() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_wait(t *testing.T) {
	t.Parallel()

	t.Run("completes after duration", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		start := time.Now()
		err := wait(ctx, 10*time.Millisecond)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("wait() error = %v, want nil", err)
		}
		if elapsed < 10*time.Millisecond {
			t.Errorf("wait() elapsed = %v, want >= 10ms", elapsed)
		}
	})

	t.Run("cancels on context done", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		err := wait(ctx, 1*time.Hour)
		if !errors.Is(err, context.Canceled) {
			t.Errorf("wait() error = %v, want context.Canceled", err)
		}
	})

	t.Run("respects context deadline", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		err := wait(ctx, 1*time.Hour)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("wait() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestController_createTag(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, dir string, name string, args ...string) error {
				capturedArgs = append([]string{dir, name}, args...)
				return nil
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.createTag(t.Context(), slog.Default(), "v1.0.0")
		if err != nil {
			t.Errorf("createTag() error = %v, want nil", err)
		}

		expectedArgs := []string{"", "git", "tag", "-m", "chore: release v1.0.0", "v1.0.0"}
		if len(capturedArgs) != len(expectedArgs) {
			t.Errorf("createTag() args = %v, want %v", capturedArgs, expectedArgs)
		}
		for i, arg := range expectedArgs {
			if capturedArgs[i] != arg {
				t.Errorf("createTag() args[%d] = %v, want %v", i, capturedArgs[i], arg)
			}
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, _ string, _ string, _ ...string) error {
				return errors.New("git error")
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.createTag(t.Context(), slog.Default(), "v1.0.0")
		if err == nil {
			t.Error("createTag() error = nil, want error")
		}
	})
}

func TestController_pushTag(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var capturedArgs []string
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, dir string, name string, args ...string) error {
				capturedArgs = append([]string{dir, name}, args...)
				return nil
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.pushTag(t.Context(), slog.Default(), "v1.0.0")
		if err != nil {
			t.Errorf("pushTag() error = %v, want nil", err)
		}

		expectedArgs := []string{"", "git", "push", "origin", "v1.0.0"}
		if diff := cmp.Diff(expectedArgs, capturedArgs); diff != "" {
			t.Errorf("args mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, _ string, _ string, _ ...string) error {
				return errors.New("git error")
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.pushTag(t.Context(), slog.Default(), "v1.0.0")
		if err == nil {
			t.Error("pushTag() error = nil, want error")
		}
	})
}

func TestController_getDefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		defaultBranch := "main"
		ghRepo := &mockRepositoriesClient{
			getFunc: func(_ context.Context, owner, repo string) (*github.Repository, *github.Response, error) {
				if owner != "test-owner" || repo != "test-repo" {
					t.Errorf("Get() called with owner=%s, repo=%s", owner, repo)
				}
				return &github.Repository{
					DefaultBranch: &defaultBranch,
				}, nil, nil
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, nil, ghRepo)

		branch, err := c.getDefaultBranch(t.Context(), slog.Default(), "test-owner", "test-repo")
		if err != nil {
			t.Errorf("getDefaultBranch() error = %v, want nil", err)
		}
		if branch != "main" {
			t.Errorf("getDefaultBranch() = %v, want main", branch)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		ghRepo := &mockRepositoriesClient{
			getFunc: func(_ context.Context, _, _ string) (*github.Repository, *github.Response, error) {
				return nil, nil, errors.New("API error")
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, nil, ghRepo)

		_, err := c.getDefaultBranch(t.Context(), slog.Default(), "owner", "repo")
		if err == nil {
			t.Error("getDefaultBranch() error = nil, want error")
		}
	})
}

func TestController_getRunID(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			outputFunc: func(_ context.Context, _ *slog.Logger, _ string, _ string, args ...string) (string, error) {
				// Verify command arguments
				if args[0] != "run" || args[1] != "list" {
					t.Errorf("unexpected args: %v", args)
				}
				return "12345", nil
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		runID, err := c.getRunID(t.Context(), slog.Default(), "release.yaml")
		if err != nil {
			t.Errorf("getRunID() error = %v, want nil", err)
		}
		if runID != "12345" {
			t.Errorf("getRunID() = %v, want 12345", runID)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			outputFunc: func(_ context.Context, _ *slog.Logger, _ string, _ string, _ ...string) (string, error) {
				return "", errors.New("gh error")
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		_, err := c.getRunID(t.Context(), slog.Default(), "release.yaml")
		if err == nil {
			t.Error("getRunID() error = nil, want error")
		}
	})
}

func TestController_watchRun(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, _ string, name string, args ...string) error {
				if name != "gh" || args[0] != "run" || args[1] != "watch" {
					t.Errorf("unexpected command: %s %v", name, args)
				}
				return nil
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.watchRun(t.Context(), slog.Default(), "12345")
		if err != nil {
			t.Errorf("watchRun() error = %v, want nil", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, _ string, _ string, _ ...string) error {
				return errors.New("workflow failed")
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.watchRun(t.Context(), slog.Default(), "12345")
		if err == nil {
			t.Error("watchRun() error = nil, want error")
		}
	})
}

func TestController_downloadArtifacts(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var capturedDir string
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, dir string, name string, args ...string) error {
				capturedDir = dir
				if diff := cmp.Diff("", dir); diff != "" {
					t.Errorf("downloadArtifacts() dir mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff("gh", name); diff != "" {
					t.Errorf("downloadArtifacts() name mismatch (-want +got):\n%s", diff)
				}
				expArgs := []string{"run", "download", "12345", "--pattern", "goreleaser", "-D", "/tmp/test"}
				if diff := cmp.Diff(expArgs, args); diff != "" {
					t.Errorf("downloadArtifacts() args mismatch (-want +got):\n%s", diff)
				}
				return nil
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.downloadArtifacts(t.Context(), slog.Default(), "/tmp/test", "12345")
		if err != nil {
			t.Errorf("downloadArtifacts() error = %v, want nil", err)
		}
		if capturedDir != "" {
			t.Errorf(`downloadArtifacts() dir = %v, want ""`, capturedDir)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()
		exec := &mockExecutor{
			runFunc: func(_ context.Context, _ *slog.Logger, _ string, _ string, _ ...string) error {
				return errors.New("download failed")
			},
		}
		c := New(afero.NewMemMapFs(), &ParamRun{}, exec, nil)

		err := c.downloadArtifacts(t.Context(), slog.Default(), "/tmp/test", "12345")
		if err == nil {
			t.Error("downloadArtifacts() error = nil, want error")
		}
	})
}
