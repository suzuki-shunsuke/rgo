package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/cmdexec"
	"github.com/suzuki-shunsuke/rgo/pkg/controller/run"
	"github.com/suzuki-shunsuke/rgo/pkg/github"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

type RunArgs struct {
	Config   string
	Workflow string
	RunID    string
	Version  string
	Publish  []string
}

func Run(ctx context.Context, logger *slogutil.Logger, env *urfave.Env) error {
	runArgs := &RunArgs{}

	return urfave.Command(env, &cli.Command{ //nolint:wrapcheck
		Name:  "rgo",
		Usage: "Release Go CLI. https://github.com/suzuki-shunsuke/rgo",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Release a Go CLI tool",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						Usage:       "Configuration file path (.goreleaser.yaml)",
						Destination: &runArgs.Config,
					},
					&cli.StringFlag{
						Name:        "workflow",
						Aliases:     []string{"w"},
						Usage:       "GitHub Actions workflow file name",
						Value:       "release.yaml",
						Destination: &runArgs.Workflow,
					},
					&cli.StringFlag{
						Name:        "run-id",
						Usage:       "GitHub Actions run ID (skip tag creation if provided)",
						Destination: &runArgs.RunID,
					},
					&cli.StringSliceFlag{
						Name:        "publish",
						Aliases:     []string{"p"},
						Usage:       "Publishers to process (homebrew, scoop, winget)",
						Destination: &runArgs.Publish,
					},
				},
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:        "version",
						Destination: &runArgs.Version,
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					return runAction(ctx, logger, cmd, runArgs)
				},
			},
		},
	}).Run(ctx, env.Args)
}

func runAction(ctx context.Context, logger *slogutil.Logger, cmd *cli.Command, args *RunArgs) error {
	if args.Version == "" {
		return errors.New("version argument is required")
	}
	param := &run.ParamRun{
		ConfigFilePath: args.Config,
		Stderr:         cmd.ErrWriter,
		Version:        args.Version,
		RunID:          args.RunID,
		Workflow:       args.Workflow,
		Publish:        args.Publish,
	}
	exec := &cmdexec.Executor{
		Stdout: cmd.Writer,
		Stderr: cmd.ErrWriter,
	}
	ghClient := github.New(ctx)
	ctrl := run.New(afero.NewOsFs(), param, exec, ghClient.Repositories)
	if err := ctrl.Run(ctx, logger.Logger); err != nil {
		return fmt.Errorf("run release: %w", err)
	}
	return nil
}
