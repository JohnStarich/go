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
	"gopkg.in/src-d/go-billy.v4"
	"gopkg.in/src-d/go-billy.v4/helper/mount"
	"gopkg.in/src-d/go-billy.v4/helper/polyfill"
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
		if err := os.RemoveAll(args.OutputPath); err != nil {
			return err
		}
		return generateDocs(modulePath, modulePackage, args)
	}

	// use temporary dir for output path, "move" files to in-memory git repo, then commit and push changes
	outputPath := args.OutputPath
	args.OutputPath, err = ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(args.OutputPath)
	if err := generateDocs(modulePath, modulePackage, args); err != nil {
		return err
	}

	repoRoot, remote, err := getCurrentPathAndRemote(modulePath)
	if err != nil {
		return err
	}

	// TODO try cloning via file path instead
	repo, err := git.Clone(memory.NewStorage(), memfs.New(), &git.CloneOptions{
		URL:           remote,
		ReferenceName: plumbing.NewBranchReferenceName(ghPagesBranch),
		SingleBranch:  true,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to clone in-memory copy of repo. Be sure the 'gh-pages' orphaned branch exists: https://help.github.com/en/github/working-with-github-pages/creating-a-github-pages-site-with-jekyll#creating-your-site")
	}

	workTree, err := repo.Worktree()
	if err != nil {
		return err
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		return errors.Wrap(err, "Failed to get absolute path of output directory")
	}
	relOutputPath, err := filepath.Rel(repoRoot, absOutputPath)
	if err != nil {
		return errors.Wrap(err, "Output path must be inside repository for gh-pages integration")
	}
	_, err = workTree.Remove(relOutputPath)
	if err != nil {
		return errors.Wrap(err, "Failed to remove output path from repo")
	}

	m := mount.New(workTree.Filesystem, filepath.Dir(args.OutputPath), osfs.New(filepath.Dir(args.OutputPath)))
	if err := renameRecursive(polyfill.New(m), args.OutputPath, relOutputPath); err != nil {
		return errors.Wrap(err, "Failed to move generated docs into repo")
	}

	if _, err := workTree.Add(relOutputPath); err != nil {
		return errors.Wrap(err, "Failed to add output dir to git")
	}

	commitMessage := "Update GoPages"
	if args.SiteTitle != "" {
		commitMessage += ": " + args.SiteTitle
	}
	_, err = workTree.Commit(commitMessage, &git.CommitOptions{
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
	return errors.Wrap(err, "Failed to push gopages commit")
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
	return repoRoot, remote.Config().URLs[0], errors.Wrap(err, "Failed to get repo remote")
}

func renameRecursive(fs billy.Filesystem, source, dest string) error {
	info, err := fs.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fs.Rename(source, dest)
	}

	if err := fs.MkdirAll(dest, info.Mode()); err != nil {
		return err
	}

	files, err := fs.ReadDir(source)
	if err != nil {
		return err
	}
	for _, f := range files {
		err := renameRecursive(fs, fs.Join(source, f.Name()), fs.Join(dest, f.Name()))
		if err != nil {
			return err
		}
	}
	return nil
}
