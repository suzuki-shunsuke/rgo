package run

import (
	"context"
	"io"
	"log/slog"
)

type ParamRun struct {
	ConfigFilePath string
	PWD            string
	Stderr         io.Writer
}

func (c *Controller) Run(ctx context.Context, logger *slog.Logger) error {
	return nil
}
