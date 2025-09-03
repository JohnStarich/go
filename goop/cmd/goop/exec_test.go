package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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

	const name = "foo"
	t.Run("invalid package name", func(t *testing.T) {
		t.Parallel()
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString("bar")
		app := newTestApp(t, testAppOptions{
			runCmd: func(_ *TestApp, cmd *exec.Cmd) error {
				return runCmd(cmd) // use real command
			},
		})
		err := app.Run([]string{"exec", "--encoded-name", encodedName, "--encoded-package", encodedPackage})
		assert.EqualError(t, err, "pipe: go install bar@latest: exit status 1")
	})

	t.Run("exec install then run", func(t *testing.T) {
		t.Parallel()
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisPackage)
		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				arg0 := strings.TrimSuffix(cmd.Args[0], path.Ext(cmd.Args[0]))
				switch arg0 {
				case "go":
					assert.Equal(t, []string{
						arg0,
						"install",
						thisPackage + "@latest",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name+systemExt(runtime.GOOS)))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case name:
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", arg0)
				}
				return nil
			},
		})
		err := app.Run([]string{"exec", "--encoded-name", encodedName, "--encoded-package", encodedPackage})
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
			{name + systemExt(runtime.GOOS)},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go"+systemExt(runtime.GOOS), filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo" + systemExt(runtime.GOOS),
		}, commandPaths)
	})

	t.Run("exec already installed run with args", func(t *testing.T) {
		t.Parallel()
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisPackage)
		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(_ *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case name:
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, app.packageInstallDir(name), 0o700))
		f, err := hackpadfs.Create(app.fs, path.Join(app.packageInstallDir(name), name+systemExt(runtime.GOOS)))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = app.Run([]string{
			"exec", "--encoded-name", encodedName, "--encoded-package", encodedPackage,
			"--", name, "-bar",
		})
		assert.NoError(t, err)

		assert.Equal(t, "", app.Stderr())
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{name, "-bar"},
		}, commandsToRun)
		assert.Equal(t, []string{
			"cache/install/foo/foo" + systemExt(runtime.GOOS),
		}, commandPaths)
	})

	t.Run("exec install local module then run", func(t *testing.T) {
		t.Parallel()
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
				arg0 := strings.TrimSuffix(cmd.Args[0], path.Ext(cmd.Args[0]))
				switch arg0 {
				case "go":
					assert.Equal(t, thisDir, cmd.Dir)
					assert.Equal(t, []string{
						"go",
						"install",
						".",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name+systemExt(runtime.GOOS)))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case name:
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", arg0)
				}
				return nil
			},
		})
		err = app.Run([]string{"exec", "--encoded-name", encodedName, "--encoded-package", encodedPackage})
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
			{name + systemExt(runtime.GOOS)},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go"+systemExt(runtime.GOOS), filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo" + systemExt(runtime.GOOS),
		}, commandPaths)
	})

	t.Run("exec local module reinstalls outdated", func(t *testing.T) {
		t.Parallel()
		const (
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
				arg0 := strings.TrimSuffix(cmd.Args[0], path.Ext(cmd.Args[0]))
				switch arg0 {
				case "go":
					assert.Equal(t, thisDir, cmd.Dir)
					assert.Equal(t, []string{
						"go",
						"install",
						".",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name+systemExt(runtime.GOOS)))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case name:
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", arg0)
				}
				return nil
			},
		})
		app.fs = newFSWithOSPath(app.fs, map[string]string{
			thisDir: workingDirFSPath,
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, workingDirFSPath, 0o700))
		require.NoError(t, hackpadfs.WriteFullFile(app.fs, path.Join(workingDirFSPath, "go.mod"), []byte("module foo"), 0o700))
		require.NoError(t, hackpadfs.WriteFullFile(app.fs, path.Join(workingDirFSPath, "main.go"), []byte(`
package main

func main() {}
`), 0o700))

		require.NoError(t, hackpadfs.MkdirAll(app.fs, app.packageInstallDir(name), 0o700))
		// set really oudated bin file that needs an update
		filePath := path.Join(app.packageInstallDir(name), name+systemExt(runtime.GOOS))
		err = hackpadfs.WriteFullFile(app.fs, filePath, nil, 0o700)
		require.NoError(t, err)
		year2000 := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		require.NoError(t, hackpadfs.Chtimes(app.fs, filePath, year2000, year2000))

		err = app.Run([]string{"exec", "--encoded-name", encodedName, "--encoded-package", encodedPackage})
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
			{name + systemExt(runtime.GOOS)},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go"+systemExt(runtime.GOOS), filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo" + systemExt(runtime.GOOS),
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

// ToOSPath implements hackpadfs.MountFS
func (fs *fsWithOSPath) ToOSPath(name string) (string, error) { //nolint:unparam // Implements interface, cannot remove param.
	for osPath, fsPath := range fs.osToFSPaths {
		if name == fsPath {
			return osPath, nil
		}
	}
	return name, nil
}

// FromOSPath implements hackpadfs.MountFS
func (fs *fsWithOSPath) FromOSPath(name string) (string, error) { //nolint:unparam // Implements interface, cannot remove param.
	fsPath, ok := fs.osToFSPaths[name]
	if !ok {
		return name, nil
	}
	return fsPath, nil
}

func writeFiles(t *testing.T, fs hackpadfs.FS, files map[string]string, modTime time.Time) {
	t.Helper()
	for filePath, contents := range files {
		require.NoError(t, hackpadfs.MkdirAll(fs, path.Dir(filePath), 0o700))
		require.NoError(t, hackpadfs.WriteFullFile(fs, filePath, []byte(contents), 0o700))
		for f := filePath; f != path.Dir(f); f = path.Dir(f) {
			require.NoError(t, hackpadfs.Chtimes(fs, f, modTime, modTime))
		}
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
			t.Parallel()
			fs, err := mem.NewFS()
			require.NoError(t, err)
			writeFiles(t, fs, tc.files, time.Now())

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

func TestHasNewerModTime(t *testing.T) {
	t.Parallel()
	var (
		now     = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
		earlier = now.Add(-time.Minute)
	)
	for _, tc := range []struct {
		description string
		files       map[string]string
		fileMtimes  map[string]time.Time
		root        string
		baseModTime time.Time
		expectNewer bool
		expectErr   string
	}{
		{
			description: "no files",
			root:        ".",
			baseModTime: now,
			expectNewer: false,
		},
		{
			description: "one newer file",
			root:        ".",
			files: map[string]string{
				"foo": "",
			},
			fileMtimes: map[string]time.Time{
				"foo": now,
			},
			baseModTime: earlier,
			expectNewer: true,
		},
		{
			description: "one older file",
			root:        ".",
			files: map[string]string{
				"foo": "",
			},
			fileMtimes: map[string]time.Time{
				"foo": earlier,
			},
			baseModTime: now,
			expectNewer: false,
		},
		{
			description: "one newer dir",
			root:        ".",
			files: map[string]string{
				"foo/bar": "",
			},
			fileMtimes: map[string]time.Time{
				"foo": now,
			},
			baseModTime: earlier,
			expectNewer: true,
		},
		{
			description: "one older dir",
			root:        ".",
			files: map[string]string{
				"foo/bar": "",
			},
			fileMtimes: map[string]time.Time{
				"foo": earlier,
			},
			baseModTime: now,
			expectNewer: false,
		},
		{
			description: "one newer before second item scanned",
			root:        ".",
			files: map[string]string{
				"bar/baz": "",
				"foo":     "",
			},
			fileMtimes: map[string]time.Time{
				"bar/baz": now,
			},
			baseModTime: earlier,
			expectNewer: true,
		},
		{
			description: "one newer with root",
			root:        "foo",
			files: map[string]string{
				"foo/bar": "",
				"foo/baz": "",
			},
			fileMtimes: map[string]time.Time{
				"foo/baz": now,
			},
			baseModTime: earlier,
			expectNewer: true,
		},
		{
			description: "none newer with different root",
			root:        "biff",
			files: map[string]string{
				"foo/bar":   "",
				"foo/baz":   "",
				"biff/derp": "",
			},
			fileMtimes: map[string]time.Time{
				"foo/baz": now,
			},
			baseModTime: earlier,
			expectNewer: false,
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			fs, err := mem.NewFS()
			require.NoError(t, err)
			writeFiles(t, fs, tc.files, earlier)
			for filePath, mtime := range tc.fileMtimes {
				require.NoError(t, hackpadfs.Chtimes(fs, filePath, mtime, mtime))
			}

			hasNewer, err := hasNewerModTime(fs, tc.root, tc.baseModTime)
			assert.Equal(t, tc.expectNewer, hasNewer)
			if tc.expectErr != "" {
				assert.EqualError(t, err, tc.expectErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
