//go:generate go run ./cmd/gendoc -template ./cmd/gendoc/doc.go -out doc.go

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

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
	"github.com/johnstarich/go/gopages/internal/pipe"
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
	switch err {
	case nil:
	case flag.ErrHelp:
		fmt.Print(usageOutput)
		return
	default:
		fmt.Print(usageOutput)
		cmd.Exit(2)
	}

	log.SetOutput(ioutil.Discard) // disable godoc's internal logging

	modulePath, err := getWD()
	if err != nil {
		panic(errors.Wrap(err, "Failed to get current directory"))
	}

	if err := runner(modulePath, args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		cmd.Exit(1)
	}
}

func run(modulePath string, args flags.Args) error {
	var modulePackage string
	var linker source.Linker
	err := pipe.ChainFuncs(
		func() error {
			var err error
			modulePackage, err = module.Package(modulePath)
			return err
		},
		func() error {
			var err error
			linker, err = args.Linker(modulePackage)
			return err
		},
	).Do()
	if err != nil {
		return err
	}

	fmt.Println("Generating godoc static pages for module...", modulePackage)
	if !args.GitHubPages {
		fs := osfs.New("")
		return generate.Docs(modulePath, modulePackage, fs, fs, args, linker)
	}

	var repoRoot, remote string
	var absOutputPath string
	err = pipe.ChainFuncs(
		func() error {
			var err error
			repoRoot, remote, err = getCurrentPathAndRemote(modulePath)
			return err
		},
		func() error {
			var err error
			absOutputPath, err = filepath.Abs(args.OutputPath)
			return errors.Wrap(err, "Failed to get absolute path of output directory")
		},
		func() error {
			var err error
			args.OutputPath, err = filepath.Rel(repoRoot, absOutputPath)
			return errors.Wrap(err, "Output path must be inside repository for gh-pages integration")
		},
	).Do()
	if err != nil {
		return err
	}

	src := osfs.New("")
	fs := memfs.New()

	var repo *git.Repository
	var workTree *git.Worktree
	err = pipe.ChainFuncs(
		func() error {
			var err error
			repo, err = git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
				URL:           remote,
				ReferenceName: plumbing.NewBranchReferenceName(ghPagesBranch),
				SingleBranch:  true,
			})
			return errors.Wrap(err, "Failed to clone in-memory copy of repo. Be sure the 'gh-pages' orphaned branch exists: https://help.github.com/en/github/working-with-github-pages/creating-a-github-pages-site-with-jekyll#creating-your-site")
		},
		func() error {
			var err error
			workTree, err = repo.Worktree()
			return err
		},
		func() error {
			_, _ = workTree.Remove(args.OutputPath) // remove old files on a best-effort basis. if the path doesn't exist, it could error
			return nil
		},
		func() error {
			return generate.Docs(modulePath, modulePackage, src, fs, args, linker)
		},
	).Do()
	if err != nil {
		return err
	}

	fmt.Println("Committing and pushing changes to gh-pages branch...")

	return pipe.ChainFuncs(
		func() error {
			_, err := workTree.Add(args.OutputPath)
			return errors.Wrap(err, "Failed to add output dir to git")
		},
		func() error {
			commitMessage := "Update GoPages"
			if args.SiteTitle != "" {
				commitMessage += ": " + args.SiteTitle
			}
			_, err := workTree.Commit(commitMessage, &git.CommitOptions{
				Author: commitAuthor(),
			})
			return errors.Wrap(err, "Failed to commit gopages files")
		},
		func() error {
			pushOpts := &git.PushOptions{}
			if args.GitHubPagesUser != "" || args.GitHubPagesToken != "" {
				pushOpts.Auth = &gitHTTP.BasicAuth{Username: args.GitHubPagesUser, Password: args.GitHubPagesToken}
			}
			err = repo.Push(pushOpts)
			return errors.Wrap(err, "Failed to push gopages commit")
		},
	).Do()
}

func getCurrentPathAndRemote(repoPath string) (string, string, error) {
	var repo *git.Repository
	var fs *git.Worktree
	var repoRoot string
	var remoteURL string
	err := pipe.ChainFuncs(
		func() error {
			var err error
			repo, err = git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
				DetectDotGit: true,
			})
			return errors.Wrapf(err, "Failed to open repo at %q", repoPath)
		},
		func() error {
			var err error
			fs, err = repo.Worktree()
			return errors.Wrap(err, "Failed to set up work tree for repo")
		},
		func() error {
			var err error
			repoRoot, err = filepath.EvalSymlinks(fs.Filesystem.Root())
			return err
		},
		func() error {
			remote, err := repo.Remote(git.DefaultRemoteName)
			remoteURL = remote.Config().URLs[0]
			return errors.Wrap(err, "Failed to get repo remote")
		},
	).Do()
	return repoRoot, remoteURL, err
}

func commitAuthor() *object.Signature {
	return &object.Signature{Name: "GoPages", When: time.Now()}
}
