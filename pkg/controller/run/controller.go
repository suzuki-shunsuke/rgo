package run

import (
	"context"
	"log/slog"

	"github.com/spf13/afero"
)

type Controller struct {
	fs    afero.Fs
	param *ParamRun
	exec  Executor
}

func New(fs afero.Fs, param *ParamRun, exec Executor) *Controller {
	return &Controller{
		param: param,
		fs:    fs,
		exec:  exec,
	}
}

type Executor interface {
	Run(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) error
	Output(ctx context.Context, logger *slog.Logger, dir string, name string, args ...string) (string, error)
}
