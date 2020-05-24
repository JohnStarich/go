package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/osfs"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gitHTTP "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
)

const (
	ghPagesBranch = "gh-pages"
)

type Args struct {
	BaseURL          string
	GitHubPages      bool
	GitHubPagesToken string
	GitHubPagesUser  string
	OutputPath       string
	SiteDescription  string
	SiteTitle        string
}

func main() {
	var args Args
	flag.StringVar(&args.OutputPath, "out", "dist", "Output path for static files")
	flag.StringVar(&args.BaseURL, "base", "", "Base URL to use for static assets")
	flag.StringVar(&args.SiteTitle, "brand-title", "", "Branding title in the top left of documentation")
	flag.StringVar(&args.SiteDescription, "brand-description", "", "Branding description in the top left of documentation")
	flag.BoolVar(&args.GitHubPages, "gh-pages", false, "Automatically commit the output path to the gh-pages branch. The current branch must be clean.")
	flag.StringVar(&args.GitHubPagesUser, "gh-pages-user", "", "The Git username to push with")
	flag.StringVar(&args.GitHubPagesToken, "gh-pages-token", "", "The Git token to push with. Usually this is an API key.")
	flag.Parse()

	log.SetOutput(ioutil.Discard) // disable godoc's internal logging

	err := run(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args Args) error {
	modulePath, err := os.Getwd()
	if err != nil {
		return err
	}

	goMod := filepath.Join(modulePath, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		return errors.New("go.mod not found in the current directory")
	}

	buf, err := ioutil.ReadFile(goMod)
	if err != nil {
		return err
	}

	modulePackage := modfile.ModulePath(buf)
	if modulePackage == "" {
		return errors.Errorf("Unable to find module package name in go.mod file: %s", goMod)
	}

	if !args.GitHubPages {
		fs := osfs.New("")
		return generateDocs(modulePath, modulePackage, args, fs, fs)
	}

	repoRoot, remote, err := getCurrentPathAndRemote(modulePath)
	if err != nil {
		return err
	}

	absOutputPath, err := filepath.Abs(args.OutputPath)
	if err != nil {
		return errors.Wrap(err, "Failed to get absolute path of output directory")
	}
	args.OutputPath, err = filepath.Rel(repoRoot, absOutputPath)
	if err != nil {
		return errors.Wrap(err, "Output path must be inside repository for gh-pages integration")
	}

	src := osfs.New("")
	fs := memfs.New()

	repo, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:           remote,
		ReferenceName: plumbing.NewBranchReferenceName(ghPagesBranch),
		SingleBranch:  true,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to clone in-memory copy of repo. Be sure the 'gh-pages' orphaned branch exists: https://help.github.com/en/github/working-with-github-pages/creating-a-github-pages-site-with-jekyll#creating-your-site")
	}

	if err := generateDocs(modulePath, modulePackage, args, src, fs); err != nil {
		return err
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return err
	}

	if _, err := workTree.Add("."); err != nil {
		return errors.Wrap(err, "Failed to add output dir to git")
	}

	commitMessage := "Update GoPages"
	if args.SiteTitle != "" {
		commitMessage += ": " + args.SiteTitle
	}
	commit, err := workTree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{Name: "GoPages", When: time.Now()},
	})
	if err != nil {
		return errors.Wrap(err, "Failed to commit gopages files")
	}

	pushOpts := &git.PushOptions{}
	if args.GitHubPagesUser != "" || args.GitHubPagesToken != "" {
		pushOpts.Auth = &gitHTTP.BasicAuth{Username: args.GitHubPagesUser, Password: args.GitHubPagesToken}
	}
	err = repo.Push(pushOpts)
	return errors.Wrapf(err, "Failed to push gopages commit %q", commit)
}

func getCurrentPathAndRemote(repoPath string) (string, string, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", "", errors.Wrapf(err, "Failed to open repo at %q", repoPath)
	}

	fs, err := repo.Worktree()
	if err != nil {
		return "", "", errors.Wrap(err, "Failed to set up work tree for repo")
	}
	repoRoot := fs.Filesystem.Root()

	remote, err := repo.Remote(git.DefaultRemoteName)
	remoteURL := remote.Config().URLs[0]
	return repoRoot, remoteURL, errors.Wrap(err, "Failed to get repo remote")
}
