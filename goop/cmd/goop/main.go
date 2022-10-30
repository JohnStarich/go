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

const appName = "goop"

func main() {
	app, err := newApp()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}

	err = app.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
}

type App struct {
	cacheDir  string
	configDir string
	errWriter io.Writer
	fs        hackpadfs.FS
	outWriter io.Writer
	runCmd    func(*exec.Cmd) error
}

func newApp() (App, error) {
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

	return App{
		cacheDir:  path.Join(cacheDir, appName),
		configDir: path.Join(configDir, appName),
		errWriter: os.Stderr,
		fs:        fs,
		outWriter: os.Stdout,
		runCmd:    runCmd,
	}, nil
}

func runCmd(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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

func (a App) fromOSPath(path string) (string, error) {
	fs, ok := a.fs.(osPathFS)
	if ok {
		return fs.FromOSPath(path)
	}
	return path, nil
}

func (a App) toOSPath(path string) (string, error) {
	fs, ok := a.fs.(osPathFS)
	if ok {
		return fs.ToOSPath(path)
	}
	return path, nil
}
