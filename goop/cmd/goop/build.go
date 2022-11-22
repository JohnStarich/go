package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/hack-pad/hackpadfs"
	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
)

func (a App) build(ctx context.Context, name string, pkg Package, alwaysBuild bool) (string, error) {
	return a.buildOS(ctx, name, pkg, alwaysBuild, runtime.GOOS)
}

func (a App) buildOS(ctx context.Context, name string, pkg Package, alwaysBuild bool, goos string) (string, error) {
	desiredPath := path.Join(a.packageInstallDir(name), name) + systemExt(goos)
	info, err := hackpadfs.Stat(a.fs, desiredPath)
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return "", err
	}
	if err == nil && info.Mode().IsRegular() && !alwaysBuild {
		shouldRebuild, err := a.shouldRebuild(info, pkg)
		if err != nil {
			return "", err
		}
		if !shouldRebuild {
			return desiredPath, nil
		}
	}

	fmt.Fprintf(a.errWriter, "Building %q...\n", pkg.Path)
	return desiredPath, a.buildAtPath(ctx, name, pkg, desiredPath)
}

func systemExt(goos string) string {
	if goos == "windows" {
		return ".exe"
	}
	return ""
}

type buildAtPathArgs struct {
	App         App
	Context     context.Context
	DesiredPath string
	InstallDir  string
	Package     Package
}

var buildAtPathPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) buildAtPathArgs {
		return args[0].(buildAtPathArgs)
	}).
	Append(func(args buildAtPathArgs) (buildAtPathArgs, error) {
		return args, hackpadfs.MkdirAll(args.App.fs, args.InstallDir, 0700)
	}).
	Append(func(args buildAtPathArgs) (buildAtPathArgs, string, error) {
		gobin, err := args.App.toOSPath(args.InstallDir)
		return args, gobin, err
	}).
	Append(func(args buildAtPathArgs, gobin string) (buildAtPathArgs, error) {
		workingDir, installPattern := args.Package.InstallPaths()
		cmdArgs := []string{"install", installPattern}
		fmt.Fprintf(args.App.errWriter, "Env: PWD=%q GOBIN=%q\nRunning 'go %s'...\n", workingDir, gobin, strings.Join(cmdArgs, " "))
		cmd := exec.CommandContext(args.Context, "go", cmdArgs...)
		cmd.Dir = workingDir
		cmd.Env = append(os.Environ(), toEnv(map[string]string{
			"GOBIN": gobin,
		})...)
		err := args.App.runCmd(cmd)
		return args, errors.WithMessage(err, formatCmd(cmd))
	}).
	Append(func(args buildAtPathArgs) (buildAtPathArgs, string, bool, error) {
		binaryPath, found, err := findBinary(args.App.fs, args.InstallDir)
		return args, binaryPath, found, err
	}).
	Append(func(args buildAtPathArgs, binaryPath string, found bool) (buildAtPathArgs, string, error) {
		return args, binaryPath, pipe.CheckError(!found,
			errors.Errorf("go install result not found at path: %s", args.InstallDir))
	}).
	Append(func(args buildAtPathArgs, binaryPath string) error {
		if binaryPath != args.DesiredPath {
			return hackpadfs.Rename(args.App.fs, binaryPath, args.DesiredPath)
		}
		return nil
	})

func (a App) buildAtPath(ctx context.Context, name string, pkg Package, desiredPath string) error {
	_, err := buildAtPathPipe.Do(buildAtPathArgs{
		App:         a,
		Context:     ctx,
		DesiredPath: desiredPath,
		InstallDir:  a.packageInstallDir(name),
		Package:     pkg,
	})
	if err == nil {
		fmt.Fprintln(a.errWriter, "Build successful.")
	}
	return errors.Unwrap(err)
}

// findBinary returns the first regular file in the directory listing
func findBinary(fs hackpadfs.FS, installDir string) (string, bool, error) {
	dirEntries, err := hackpadfs.ReadDir(fs, installDir)
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return "", false, err
	}
	for _, entry := range dirEntries {
		if entry.Type().IsRegular() {
			return path.Join(installDir, entry.Name()), true, nil
		}
	}
	return "", false, nil
}

func (a App) shouldRebuild(binaryInfo hackpadfs.FileInfo, pkg Package) (bool, error) {
	filePath, isFilePath := pkg.FilePath()
	if !isFilePath {
		// remote module, assume up-to-date
		return false, nil
	}

	// local module, find out if the build is stale
	binaryModTime := binaryInfo.ModTime()
	fsPath, err := a.fromOSPath(filePath)
	if err != nil {
		return false, err
	}
	moduleRoot, err := moduleRoot(a.fs, fsPath)
	if err != nil {
		return false, err
	}
	return hasNewerModTime(a.fs, moduleRoot, binaryModTime)
}

func moduleRoot(fs hackpadfs.FS, p string) (string, error) {
	const goMod = "go.mod"
	_, err := hackpadfs.Stat(fs, path.Join(p, goMod))
	if err == nil {
		return p, nil
	}
	parentPath := path.Dir(p)
	if parentPath == p {
		return "", errors.Errorf("go.mod not found for package: %q", p)
	}
	root, parentErr := moduleRoot(fs, parentPath)
	if parentErr != nil {
		// if parent also failed to find module, return original error
		return "", err
	}
	return root, nil
}

func hasNewerModTime(fs hackpadfs.FS, root string, baseModTime time.Time) (bool, error) {
	hasNewerModTime := false
	err := hackpadfs.WalkDir(fs, root, func(path string, d hackpadfs.DirEntry, err error) error {
		if hasNewerModTime {
			return hackpadfs.SkipDir
		}
		if err != nil || path == root {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		modTime := info.ModTime()
		if modTime.After(baseModTime) {
			hasNewerModTime = true
			return hackpadfs.SkipDir
		}
		return nil
	})
	return hasNewerModTime, err
}
