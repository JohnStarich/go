package main

import (
	"fmt"
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
	cacheDir   string
	configDir  string
	fromOSPath func(string) (string, error)
	fs         hackpadfs.FS
	getenv     func(string) string
	runCmd     func(*exec.Cmd) error
	toOSPath   func(string) (string, error)
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
		cacheDir:   path.Join(cacheDir, appName),
		configDir:  path.Join(configDir, appName),
		fromOSPath: fs.FromOSPath,
		fs:         fs,
		getenv:     os.Getenv,
		runCmd:     runCmd,
		toOSPath:   fs.ToOSPath,
	}, nil
}

func runCmd(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (a App) Run(args []string) error {
	moduleFlag := &cli.StringFlag{
		Name:     "module",
		Required: true,
		Aliases:  []string{"m"},
	}
	cliApp := &cli.App{
		Name: appName,
		Commands: []*cli.Command{
			{
				Name:   "info",
				Before: a.noArgs,
				Action: a.info,
			},
			{
				Name:   "install",
				Before: a.noArgs,
				Action: a.install,
				Flags: []cli.Flag{
					moduleFlag,
				},
			},
			{
				Name:   "exec",
				Hidden: true,
				Action: a.exec,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "encoded-module",
						Required: true,
					},
				},
			},
			{
				Name:   "rm",
				Before: a.noArgs,
				Action: a.rm,
				Flags: []cli.Flag{
					moduleFlag,
				},
			},
		},
		ExitErrHandler: func(*cli.Context, error) {},
	}
	return cliApp.Run(args)
}

func (a App) noArgs(c *cli.Context) error {
	if c.NArg() > 0 {
		return fmt.Errorf("unexpected arguments used without flags: %s", strings.Join(c.Args().Slice(), " "))
	}
	return nil
}
