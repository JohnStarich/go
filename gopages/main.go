//go:generate go run ./cmd/gendoc -template ./cmd/gendoc/doc.go -out doc.go

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	gitHTTP "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/johnstarich/go/gopages/cmd"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/johnstarich/go/gopages/internal/generate"
	"github.com/johnstarich/go/gopages/internal/generate/source"
	"github.com/johnstarich/go/gopages/internal/module"
	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
)

const (
	ghPagesBranch = "gh-pages"
)

func main() {
	mainArgs(run, os.Getwd, os.Args[1:]...)
}

func mainArgs(
	runner func(string, flags.Args) error,
	getWD func() (string, error),
	osArgs ...string,
) {
	args, usageOutput, err := flags.Parse(osArgs...)
	switch {
	case err == nil:
	case errors.Is(err, flag.ErrHelp):
		fmt.Print(usageOutput)
		return
	default:
		fmt.Print(usageOutput)
		cmd.Exit(cmd.ExitCodeInvalidUsage)
	}

	log.SetOutput(io.Discard) // disable godoc's internal logging

	modulePath, err := getWD()
	if err != nil {
		panic(errors.Wrap(err, "Failed to get current directory"))
	}

	if err := runner(modulePath, args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		cmd.Exit(1)
	}
}

var makeLinkerPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) (flags.Args, string) {
		flagArgs := args[0].(flags.Args)
		modulePath := args[1].(string)
		return flagArgs, modulePath
	}).
	Append(func(args flags.Args, modulePath string) (flags.Args, string, error) {
		modulePackage, err := module.Package(modulePath)
		return args, modulePackage, err
	}).
	Append(func(args flags.Args, modulePackage string) (string, source.Linker, error) {
		linker, err := args.Linker(modulePackage)
		return modulePackage, linker, err
	})

var findOutputPathPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) (flags.Args, string) {
		flagArgs := args[0].(flags.Args)
		modulePath := args[1].(string)
		return flagArgs, modulePath
	}).
	Append(func(args flags.Args, modulePath string) (flags.Args, string, string, error) {
		repoRoot, remote, err := getCurrentPathAndRemote(modulePath)
		return args, repoRoot, remote, err
	}).
	Append(func(args flags.Args, repoRoot, remote string) (flags.Args, string, string, string, error) {
		absOutputPath, err := filepath.Abs(args.OutputPath)
		return args, repoRoot, absOutputPath, remote, errors.Wrap(err, "Failed to get absolute path of output directory")
	}).
	Append(func(args flags.Args, repoRoot, absOutputPath, remote string) (flags.Args, string, error) {
		var err error
		args.OutputPath, err = filepath.Rel(repoRoot, absOutputPath)
		return args, remote, errors.Wrap(err, "Output path must be inside repository for gh-pages integration")
	})

type memfsDocsArgs struct {
	Flags         flags.Args
	Linker        source.Linker
	ModulePackage string
	ModulePath    string
	Remote        string
	SrcFS, FS     billy.Filesystem
}

var generateMemfsDocsPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) memfsDocsArgs {
		return args[0].(memfsDocsArgs)
	}).
	Append(func(args memfsDocsArgs) (memfsDocsArgs, *git.Repository, error) {
		repo, err := git.Clone(memory.NewStorage(), args.FS, &git.CloneOptions{
			URL:           args.Remote,
			ReferenceName: plumbing.NewBranchReferenceName(ghPagesBranch),
			SingleBranch:  true,
			Auth:          getAuth(args.Flags),
		})
		return args, repo, errors.Wrap(err, "Failed to clone in-memory copy of repo. Be sure the 'gh-pages' orphaned branch exists: https://help.github.com/en/github/working-with-github-pages/creating-a-github-pages-site-with-jekyll#creating-your-site")
	}).
	Append(func(args memfsDocsArgs, repo *git.Repository) (memfsDocsArgs, *git.Repository, *git.Worktree, error) {
		workTree, err := repo.Worktree()
		return args, repo, workTree, err
	}).
	Append(func(args memfsDocsArgs, repo *git.Repository, workTree *git.Worktree) (memfsDocsArgs, *git.Repository, *git.Worktree, error) {
		_, _ = workTree.Remove(args.Flags.OutputPath) // remove old files on a best-effort basis. if the path doesn't exist, it could error
		err := generate.Docs(args.ModulePath, args.ModulePackage, args.SrcFS, args.FS, args.Flags, args.Linker)
		return args, repo, workTree, err
	}).
	Append(func(args memfsDocsArgs, repo *git.Repository, workTree *git.Worktree) (memfsDocsArgs, *git.Repository, *git.Worktree, error) {
		fmt.Println("Committing and pushing changes to gh-pages branch...")
		_, err := workTree.Add(args.Flags.OutputPath)
		return args, repo, workTree, errors.Wrap(err, "Failed to add output dir to git")
	}).
	Append(func(args memfsDocsArgs, repo *git.Repository, workTree *git.Worktree) (memfsDocsArgs, *git.Repository, error) {
		commitMessage := "Update GoPages"
		if args.Flags.SiteTitle != "" {
			commitMessage += ": " + args.Flags.SiteTitle
		}
		_, err := workTree.Commit(commitMessage, &git.CommitOptions{
			Author: commitAuthor(),
		})
		return args, repo, errors.Wrap(err, "Failed to commit gopages files")
	}).
	Append(func(args memfsDocsArgs, repo *git.Repository) (memfsDocsArgs, *git.Repository, *url.URL, error) {
		remoteURL, err := url.Parse(args.Remote)
		return args, repo, remoteURL, err
	}).
	Append(func(args memfsDocsArgs, repo *git.Repository, remoteURL *url.URL) error {
		remoteURL.User = nil
		pushOpts := &git.PushOptions{RemoteURL: remoteURL.String()}
		pushOpts.Auth = getAuth(args.Flags)
		err := repo.Push(pushOpts)
		return errors.Wrap(err, "Failed to push gopages commit")
	})

func getAuth(args flags.Args) *gitHTTP.BasicAuth {
	var auth *gitHTTP.BasicAuth
	if args.GitHubPagesUser != "" || args.GitHubPagesToken != "" {
		auth = &gitHTTP.BasicAuth{Username: args.GitHubPagesUser, Password: args.GitHubPagesToken}
	}
	return auth
}

func run(modulePath string, args flags.Args) error {
	out, err := makeLinkerPipe.Do(args, modulePath)
	if err != nil {
		return err
	}
	modulePackage := out[0].(string)
	linker := out[1].(source.Linker)

	fmt.Println("Generating godoc static pages for module...", modulePackage)
	if !args.GitHubPages {
		fs := osfs.New("")
		return generate.Docs(modulePath, modulePackage, fs, fs, args, linker)
	}

	out, err = findOutputPathPipe.Do(args, modulePath)
	if err != nil {
		return err
	}
	args = out[0].(flags.Args)
	remote := out[1].(string)

	_, err = generateMemfsDocsPipe.Do(memfsDocsArgs{
		Flags:         args,
		Linker:        linker,
		ModulePackage: modulePackage,
		ModulePath:    modulePath,
		Remote:        remote,
		SrcFS:         osfs.New(""),
		FS:            memfs.New(),
	})
	return err
}

var currentPathAndRemotePipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) string {
		return args[0].(string)
	}).
	Append(func(repoPath string) (*git.Repository, error) {
		repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
			DetectDotGit: true,
		})
		return repo, errors.Wrapf(err, "Failed to open repo at %q", repoPath)
	}).
	Append(func(repo *git.Repository) (*git.Repository, *git.Worktree, error) {
		workTree, err := repo.Worktree()
		return repo, workTree, errors.Wrap(err, "Failed to set up work tree for repo")
	}).
	Append(func(repo *git.Repository, workTree *git.Worktree) (*git.Repository, string, error) {
		repoRoot, err := filepath.EvalSymlinks(workTree.Filesystem.Root())
		return repo, repoRoot, err
	}).
	Append(func(repo *git.Repository, repoRoot string) (string, string, error) {
		remote, err := repo.Remote(git.DefaultRemoteName)
		remoteURL := remote.Config().URLs[0]
		return repoRoot, remoteURL, errors.Wrap(err, "Failed to get repo remote")
	})

func getCurrentPathAndRemote(repoPath string) (string, string, error) {
	out, err := currentPathAndRemotePipe.Do(repoPath)
	var repoRoot, remoteURL string
	if err == nil {
		repoRoot = out[0].(string)
		remoteURL = out[1].(string)
	}
	return repoRoot, remoteURL, err
}

func commitAuthor() *object.Signature {
	return &object.Signature{Name: "GoPages", When: time.Now()}
}
