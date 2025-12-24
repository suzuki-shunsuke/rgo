package cli

import (
	"context"
	"os"

	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/cmdexec"
	"github.com/suzuki-shunsuke/rgo/pkg/controller/run"
	"github.com/suzuki-shunsuke/slog-util/slogutil"
	"github.com/suzuki-shunsuke/urfave-cli-v3-util/urfave"
	"github.com/urfave/cli/v3"
)

func Run(ctx context.Context, logger *slogutil.Logger, env *urfave.Env) error {
	return urfave.Command(env, &cli.Command{ //nolint:wrapcheck
		Name:  "rgo",
		Usage: "Release Go CLI. https://github.com/suzuki-shunsuke/rgo",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Release a Go CLI tool",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Usage:   "Configuration file path (.goreleaser.yaml)",
					},
					&cli.StringFlag{
						Name:    "workflow",
						Aliases: []string{"w"},
						Usage:   "GitHub Actions workflow file name",
						Value:   "release.yaml",
					},
					&cli.StringFlag{
						Name:  "run-id",
						Usage: "GitHub Actions run ID (skip tag creation if provided)",
					},
				},
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "version",
						UsageText: "version to release (e.g., v1.0.0)",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					version := cmd.Args().First()
					if version == "" {
						return cli.Exit("version argument is required", 1)
					}
					pwd, err := os.Getwd()
					if err != nil {
						return err
					}
					param := &run.ParamRun{
						ConfigFilePath: cmd.String("config"),
						PWD:            pwd,
						Stderr:         cmd.ErrWriter,
						Version:        version,
						RunID:          cmd.String("run-id"),
						Workflow:       cmd.String("workflow"),
					}
					exec := &cmdexec.Executor{
						Stdout: cmd.Writer,
						Stderr: cmd.ErrWriter,
					}
					ctrl := run.New(afero.NewOsFs(), param, exec)
					return ctrl.Run(ctx, logger.Logger)
				},
			},
		},
	}).Run(ctx, env.Args)
}
