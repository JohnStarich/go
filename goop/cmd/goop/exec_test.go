package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const thisPackage = "github.com/johnstarich/go/goop/cmd/goop"

func TestExec(t *testing.T) {
	t.Parallel()

	t.Run("invalid package name", func(t *testing.T) {
		encodedName := base64EncodeString("foo")
		encodedPackage := base64EncodeString("bar")
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				return runCmd(cmd) // use real command
			},
		})
		err := app.Run([]string{"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage})
		assert.EqualError(t, err, "pipe: go install bar@latest: exit status 1")
	})

	t.Run("exec install then run", func(t *testing.T) {
		const name = "foo"
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisPackage)
		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case "go":
					assert.Equal(t, []string{
						"go",
						"install",
						thisPackage + "@latest",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case "foo":
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		err := app.Run([]string{"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(`
Building "github.com/johnstarich/go/goop/cmd/goop"...
Env: PWD="" GOBIN="cache/install/foo"
Running 'go install github.com/johnstarich/go/goop/cmd/goop@latest'...
Build successful.
`), strings.TrimSpace(app.Stderr()))
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", thisPackage + "@latest"},
			{"foo"},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go", filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo",
		}, commandPaths)
	})

	t.Run("exec already installed run with args", func(t *testing.T) {
		const name = "foo"
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisPackage)
		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(_ *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case "foo":
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, app.packageInstallDir(name), 0700))
		f, err := hackpadfs.Create(app.fs, path.Join(app.packageInstallDir(name), name))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = app.Run([]string{
			"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage,
			"--", name, "-bar",
		})
		assert.NoError(t, err)

		assert.Equal(t, "", app.Stderr())
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{name, "-bar"},
		}, commandsToRun)
		assert.Equal(t, []string{
			"cache/install/foo/foo",
		}, commandPaths)
	})

	t.Run("exec install local module then run", func(t *testing.T) {
		const name = "foo"
		thisDir, err := os.Getwd()
		require.NoError(t, err)
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisDir)

		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case "go":
					assert.Equal(t, thisDir, cmd.Dir)
					assert.Equal(t, []string{
						"go",
						"install",
						".",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case "foo":
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		err = app.Run([]string{"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(fmt.Sprintf(`
Building %q...
Env: PWD=%q GOBIN="cache/install/foo"
Running 'go install .'...
Build successful.
`, thisDir, thisDir)), strings.TrimSpace(app.Stderr()))
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", "."},
			{"foo"},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go", filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo",
		}, commandPaths)
	})

	t.Run("exec local module reinstalls outdated", func(t *testing.T) {
		const (
			name             = "foo"
			workingDirFSPath = "some/working/directory"
		)
		thisDir, err := os.Getwd()
		require.NoError(t, err)
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisDir)

		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case "go":
					assert.Equal(t, thisDir, cmd.Dir)
					assert.Equal(t, []string{
						"go",
						"install",
						".",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case "foo":
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		app.fs = newFSWithOSPath(app.fs, map[string]string{
			thisDir: workingDirFSPath,
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, workingDirFSPath, 0700))
		require.NoError(t, hackpadfs.WriteFullFile(app.fs, path.Join(workingDirFSPath, "go.mod"), []byte("module foo"), 0700))
		require.NoError(t, hackpadfs.WriteFullFile(app.fs, path.Join(workingDirFSPath, "main.go"), []byte(`
package main

func main() {}
`), 0700))

		require.NoError(t, hackpadfs.MkdirAll(app.fs, app.packageInstallDir(name), 0700))
		// set really oudated bin file that needs an update
		filePath := path.Join(app.packageInstallDir(name), name)
		err = hackpadfs.WriteFullFile(app.fs, filePath, nil, 0700)
		require.NoError(t, err)
		year2000 := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		require.NoError(t, hackpadfs.Chtimes(app.fs, filePath, year2000, year2000))

		err = app.Run([]string{"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(fmt.Sprintf(`
Building %q...
Env: PWD=%q GOBIN="cache/install/foo"
Running 'go install .'...
Build successful.
`, thisDir, thisDir)), strings.TrimSpace(app.Stderr()))
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", "."},
			{name},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go", filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo",
		}, commandPaths)
	})
}

func fromEnv(env []string) map[string]string {
	m := make(map[string]string)
	for _, envPair := range env {
		equalIndex := strings.IndexRune(envPair, '=')
		if equalIndex == -1 {
			panic("Invalid env key-value pair: " + envPair)
		}
		key, value := envPair[:equalIndex], envPair[equalIndex+1:]
		m[key] = value
	}
	return m
}

func base64EncodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

type fsWithOSPath struct {
	hackpadfs.FS
	osToFSPaths map[string]string
}

func newFSWithOSPath(fs hackpadfs.FS, osToFSPaths map[string]string) *fsWithOSPath {
	return &fsWithOSPath{
		FS:          fs,
		osToFSPaths: osToFSPaths,
	}
}

func (fs *fsWithOSPath) Mount(name string) (hackpadfs.FS, string) {
	return fs.FS, name
}

func (fs *fsWithOSPath) ToOSPath(name string) (string, error) {
	for osPath, fsPath := range fs.osToFSPaths {
		if name == fsPath {
			return osPath, nil
		}
	}
	return name, nil
}

func (fs *fsWithOSPath) FromOSPath(name string) (string, error) {
	fsPath, ok := fs.osToFSPaths[name]
	if !ok {
		return name, nil
	}
	return fsPath, nil
}

func writeFiles(t *testing.T, fs hackpadfs.FS, files map[string]string) {
	for filePath, contents := range files {
		require.NoError(t, hackpadfs.MkdirAll(fs, path.Dir(filePath), 0700))
		require.NoError(t, hackpadfs.WriteFullFile(fs, filePath, []byte(contents), 0700))
	}
}

func TestModuleRoot(t *testing.T) {
	t.Parallel()
	const (
		someModuleFile = "module foo"
		someGoFile     = "package main"
	)
	for _, tc := range []struct {
		description string
		files       map[string]string
		path        string
		expectRoot  string
		expectErr   string
	}{
		{
			description: "no files",
			path:        "",
			expectErr:   "stat go.mod: file does not exist",
		},
		{
			description: "no files with root path",
			path:        ".",
			expectErr:   "go.mod not found for package: \".\"",
		},
		{
			description: "no files with path",
			path:        "foo",
			expectErr:   "stat foo/go.mod: file does not exist",
		},
		{
			description: "same root as path",
			files: map[string]string{
				"foo/go.mod": someModuleFile,
			},
			path:       "foo",
			expectRoot: "foo",
		},
		{
			description: "same root as path nested",
			files: map[string]string{
				"foo/bar/go.mod": someModuleFile,
				"foo/go.mod":     someModuleFile,
			},
			path:       "foo/bar",
			expectRoot: "foo/bar",
		},
		{
			description: "root one up",
			files: map[string]string{
				"foo/bar/main.go": someGoFile,
				"foo/go.mod":      someModuleFile,
			},
			path:       "foo/bar",
			expectRoot: "foo",
		},
		{
			description: "root root",
			files: map[string]string{
				"go.mod": someModuleFile,
			},
			path:       ".",
			expectRoot: ".",
		},
		{
			description: "root root with path",
			files: map[string]string{
				"foo/bar/main.go": someGoFile,
				"go.mod":          someModuleFile,
			},
			path:       "foo/bar",
			expectRoot: ".",
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			fs, err := mem.NewFS()
			require.NoError(t, err)
			writeFiles(t, fs, tc.files)

			root, err := moduleRoot(fs, tc.path)
			assert.Equal(t, tc.expectRoot, root)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
