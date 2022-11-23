// Command goop installs and manages both local and remote Go modules.
package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/hack-pad/hackpadfs"
	osfs "github.com/hack-pad/hackpadfs/os"
	"github.com/spf13/cobra"
)

func run(args []string, outWriter, errWriter io.Writer) error {
	app, err := newApp(outWriter, errWriter)
	if err == nil {
		err = app.Run(args[1:])
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
// Args should not include arg0.
func (a App) Run(args []string) error {
	ranCommand := false
	rootCommand := &cobra.Command{
		Use: appName,
		Long: strings.TrimSpace(`
Write and run Go scripts without the fuss. Start with 'goop install', then run the module's name as a command to execute it.

Includes:
- Support for both local and remote modules.
- Automatic rebuilds of local modules.
- Shareable command bin for easy setup on multiple machines.
`),
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			cmd.SilenceUsage = true // See https://github.com/spf13/cobra/issues/340#issuecomment-378726225
			ranCommand = true
		},
	}

	rootCommand.SetOut(a.outWriter)
	rootCommand.SetErr(a.errWriter)
	rootCommand.SetHelpCommand(&cobra.Command{Hidden: true}) // remove help subcommand

	infoCommand := &cobra.Command{
		Use:   "info",
		Short: "Shows general information for currently installed modules.",
		RunE:  a.info,
	}
	rootCommand.AddCommand(infoCommand)

	installCommand := &cobra.Command{
		Use:   "install",
		Short: "Installs modules as commands for use on the command-line.",
		Long: `Installs modules as commands for use on the command-line.

For example, run 'goop install -p github.com/johnstarich/go/covet/cmd/covet' to build and install the covet tool, then run 'covet --help' to execute the covet command.

To run an installed module, use its name on the command-line. For local modules, Goop automatically triggers a rebuild when the command is out of date. This means local scripts can be updated and used immediately.

Set the GOOP_BIN environment variable to select a custom command location. This is helpful when sharing commands across multiple machines with a tool like OneDrive, iCloud Drive, or Google Drive.`,
		RunE: a.install,
	}
	rootCommand.AddCommand(installCommand)
	installCommand.Flags().StringP("package", "p", "", "The package pattern to install. Can be a local or remote module. Remote modules may use a '@version' like '-p github.com/johnstarich/go/covet/cmd/covet@latest'. Local modules must use absolute paths without a '@version' like '-p /path/to/my/module'.")
	panicIfErr(installCommand.MarkFlagRequired("package"))
	installCommand.Flags().String("name", "", "An optional name for the command when installed. For example, 'goop install -p github.com/johnstarich/go/covet/cmd/covet -name foo' and then run 'foo' as the command. Defaults to the package base name.")

	removeCommand := &cobra.Command{
		Use:   "rm",
		Short: "Removes a previously installed command.",
		RunE:  a.rm,
	}
	rootCommand.AddCommand(removeCommand)
	removeCommand.Flags().String("name", "", "The name of the module command to remove.")
	panicIfErr(removeCommand.MarkFlagRequired("name"))

	applyCommands(rootCommand.Commands(), func(cmd *cobra.Command) {
		cmd.Args = cobra.NoArgs
	})
	// Add exec command after applying 'NoArgs'.
	execCommand := &cobra.Command{
		Use:    "exec",
		Hidden: true,
		RunE:   a.exec,
	}
	rootCommand.AddCommand(execCommand)
	execCommand.Flags().String("encoded-name", "", "")
	panicIfErr(execCommand.MarkFlagRequired("encoded-name"))
	execCommand.Flags().String("encoded-package", "", "")
	panicIfErr(execCommand.MarkFlagRequired("encoded-package"))

	rootCommand.SetArgs(args)
	err := rootCommand.ExecuteContext(context.Background())
	if !ranCommand {
		const usageExitCode = 2
		err = wrapExitCode(err, usageExitCode)
	}
	return err
}

// panicIfErr is used to panic on any error which should be impossible.
// Helps test both nil and non-nil error branches.
func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

type exitCodeError struct {
	err  error
	code int
}

func wrapExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return exitCodeError{
		err:  err,
		code: code,
	}
}

func (e exitCodeError) Error() string {
	return e.err.Error()
}

func (e exitCodeError) Unwrap() error {
	return e.err
}

func (e exitCodeError) ExitCode() int {
	return e.code
}

func applyCommands(commands []*cobra.Command, apply func(cmd *cobra.Command)) {
	for _, cmd := range commands {
		apply(cmd)
		applyCommands(cmd.Commands(), apply)
	}
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
