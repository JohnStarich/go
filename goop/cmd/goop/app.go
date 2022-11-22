// Command goop installs and manages both local and remote Go modules.
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

// App is the goop application object, used to run commands from user input.
type App struct {
	errWriter       io.Writer
	fs              hackpadfs.FS
	getEnv          func(string) string
	lookPath        func(string) (string, error)
	outWriter       io.Writer
	runCmd          func(*exec.Cmd) error
	staticBinDir    string
	staticCacheDir  string
	staticOSHomeDir string
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
	osHomeDir, err := os.UserHomeDir()
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
		errWriter:       errWriter,
		fs:              fs,
		getEnv:          os.Getenv,
		lookPath:        exec.LookPath,
		outWriter:       outWriter,
		runCmd:          runCmd,
		staticBinDir:    path.Join(configDir, appName, configBin),
		staticCacheDir:  path.Join(cacheDir, appName),
		staticOSHomeDir: osHomeDir,
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

// Run executes this application with the given CLI args.
func (a App) Run(args []string) error {
	cliApp := &cli.App{
		Name: appName,
		Usage: strings.TrimSpace(`
Write and run Go scripts without the fuss. Start with 'goop install', then run the module's name as a command to execute it.

Includes:
- Support for both local and remote modules.
- Automatic rebuilds of local modules.
- Shareable command bin for easy setup on multiple machines.
`),
		Commands: []*cli.Command{
			{
				Name:   "info",
				Usage:  "Shows general information for currently installed modules.",
				Action: a.info,
			},
			{
				Name:  "install",
				Usage: "Installs modules as commands for use on the command-line.",
				Description: `Installs modules as commands for use on the command-line.

For example, run 'goop install -p github.com/johnstarich/go/covet/cmd/covet' to build and install the covet tool, then run 'covet --help' to execute the covet command.

To run an installed module, use its name on the command-line. For local modules, Goop automatically triggers a rebuild when the command is out of date. This means local scripts can be updated and used immediately.

Set the GOOP_BIN environment variable to select a custom command location. This is helpful when sharing commands across multiple machines with a tool like OneDrive, iCloud Drive, or Google Drive.`,
				Action: a.install,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "package",
						Usage:    "The package pattern to install. Can be a local or remote module. Remote modules may use a '@version' like '-p github.com/johnstarich/go/covet/cmd/covet@latest'. Local modules must use absolute paths without a '@version' like '-p /path/to/my/module'.",
						Required: true,
						Aliases:  []string{"p"},
					},
					&cli.StringFlag{
						Name:  "name",
						Usage: "An optional name for the command when installed. For example, 'goop install -p github.com/johnstarich/go/covet/cmd/covet -name foo' and then run 'foo' as the command. Defaults to the package base name.",
					},
				},
			},
			{
				Name:   "rm",
				Usage:  "Removes a previously installed command.",
				Action: a.rm,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "name",
						Usage:    "The name of the module command to remove.",
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
