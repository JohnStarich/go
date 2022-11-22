package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/hack-pad/hackpadfs"
	osfs "github.com/hack-pad/hackpadfs/os"
	"github.com/urfave/cli/v2"
)

func run(args []string, outWriter, errWriter io.Writer) error {
	app, err := newApp(outWriter, errWriter)
	if err == nil {
		err = app.Run(args)
	}
	return err
}

const appName = "goop"

type App struct {
	errWriter      io.Writer
	fs             hackpadfs.FS
	getEnv         func(string) string
	outWriter      io.Writer
	runCmd         func(*exec.Cmd) error
	staticBinDir   string
	staticCacheDir string
}

func newApp(outWriter, errWriter io.Writer) (App, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return App{}, err
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return App{}, err
	}

	// convert OS paths to FS paths
	osPaths := []*string{
		&configDir,
		&cacheDir,
	}
	fs := osfs.NewFS()
	for _, p := range osPaths {
		*p, err = fs.FromOSPath(*p)
		if err != nil {
			return App{}, err
		}
	}

	const configBin = "bin"
	return App{
		errWriter:      errWriter,
		fs:             fs,
		getEnv:         os.Getenv,
		outWriter:      outWriter,
		runCmd:         runCmd,
		staticBinDir:   path.Join(configDir, appName, configBin),
		staticCacheDir: path.Join(cacheDir, appName),
	}, nil
}

func runCmd(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func formatCmd(cmd *exec.Cmd) string {
	return strings.Join(cmd.Args, " ")
}

func (a App) Run(args []string) error {
	cliApp := &cli.App{
		Name: appName,
		Commands: []*cli.Command{
			{
				Name:   "info",
				Action: a.info,
			},
			{
				Name:   "install",
				Action: a.install,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "package",
						Required: true,
						Aliases:  []string{"p"},
					},
					&cli.StringFlag{
						Name: "name",
					},
				},
			},
			{
				Name:   "rm",
				Action: a.rm,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Required: true,
					},
				},
			},
		},
		HideHelpCommand: true,
		ErrWriter:       a.errWriter,
		ExitErrHandler:  func(*cli.Context, error) {},
		Writer:          a.outWriter,
	}

	applyCommands(cliApp.Commands, func(cmd *cli.Command) {
		cmd.Before = a.noArgs
	})
	// Add exec command after applying 'Before's.
	cliApp.Commands = append(cliApp.Commands, &cli.Command{
		Name:   "exec",
		Hidden: true,
		Action: a.exec,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "encoded-name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "encoded-package",
				Required: true,
			},
		},
	})
	return cliApp.Run(args)
}

func applyCommands(commands cli.Commands, apply func(cmd *cli.Command)) {
	for _, cmd := range commands {
		apply(cmd)
		applyCommands(cmd.Subcommands, apply)
	}
}

func (a App) noArgs(c *cli.Context) error {
	if c.NArg() > 0 {
		return fmt.Errorf("unexpected arguments used without flags: %s", strings.Join(c.Args().Slice(), " "))
	}
	return nil
}

type osPathFS interface {
	hackpadfs.FS
	FromOSPath(path string) (string, error)
	ToOSPath(path string) (string, error)
}

// fromOSPath attempts to derive the FS path from an OS-like path
func (a App) fromOSPath(path string) (string, error) {
	fs, ok := a.fs.(osPathFS)
	if ok {
		return fs.FromOSPath(path)
	}
	return path, nil
}

// toOSPath attempts to derive an OS path from an FS path
func (a App) toOSPath(path string) (string, error) {
	fs, ok := a.fs.(osPathFS)
	if ok {
		return fs.ToOSPath(path)
	}
	return path, nil
}
