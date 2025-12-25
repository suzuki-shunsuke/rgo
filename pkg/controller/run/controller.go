package run

import (
	"context"
	"log/slog"

	"github.com/google/go-github/v80/github"
	"github.com/spf13/afero"
)

type Controller struct {
	fs     afero.Fs
	param  *ParamRun
	exec   Executor
	ghRepo RepositoriesClient
}

func New(fs afero.Fs, param *ParamRun, exec Executor, ghRepo RepositoriesClient) *Controller {
	return &Controller{
		param:  param,
		fs:     fs,
		exec:   exec,
		ghRepo: ghRepo,
	}
}

type Executor interface {
	Run(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error
	Output(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) (string, error)
}

type RepositoriesClient interface {
	Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
}
