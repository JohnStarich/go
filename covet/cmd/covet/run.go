package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/os"
	"github.com/johnstarich/go/covet"
	"github.com/johnstarich/go/covet/internal/coverstatus"
	"github.com/johnstarich/go/covet/internal/span"
	"github.com/johnstarich/go/covet/internal/summary"
	"github.com/pkg/errors"
)

const maxPercentInt = 100

// Args contains all flag values for a covet run
type Args struct {
	DiffFile           string
	DiffBaseDir        string
	GoCoverageFile     string
	ShowCoverage       bool
	TargetDiffCoverage uint

	GitHubToken    string
	GitHubIssue    string
	GitHubEndpoint string
}

func run(
	strArgs []string,
	stdin io.Reader,
	stdout,
	stderr io.Writer,
	fs hackpadfs.FS,
) error {
	args, err := parseArgs(strArgs, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			err = nil
		}
		return err
	}
	deps := Deps{
		Stdin:  stdin,
		Stdout: stdout,
		FS:     fs,
	}
	return runArgs(args, deps)
}

func parseArgs(strArgs []string, output io.Writer) (Args, error) {
	const defaultTargetDiffCov = 90
	var args Args
	set := flag.NewFlagSet("covet", flag.ContinueOnError)
	set.SetOutput(output)
	set.StringVar(&args.DiffFile, "diff-file", "", "Required. Path to a diff file. Use '-' for stdin.")
	set.StringVar(&args.DiffBaseDir, "diff-base-dir", ".", "Path to the diff's base directory. Defaults to the current directory.")
	set.StringVar(&args.GoCoverageFile, "cover-go", "", "Required. Path to a Go coverage profile.")
	set.BoolVar(&args.ShowCoverage, "show-diff-coverage", false, "Show the coverage diff in addition to the summary.")
	set.UintVar(&args.TargetDiffCoverage, "target-diff-coverage", defaultTargetDiffCov, "Target total test coverage of new lines. Reports the biggest gaps needed to reach the target. Any number between 0 and 100.")
	set.StringVar(&args.GitHubToken, "gh-token", "", "GitHub access token to post and update a PR comment. If running in GitHub Actions, a comment may not be necessary.")
	set.StringVar(&args.GitHubEndpoint, "gh-api", "https://api.github.com", "GitHub API endpoint. Required for GitHub Enterprise.")
	set.StringVar(&args.GitHubIssue, "gh-issue", "", "GitHub issue or pull request URL. Example: github.com/org/repo/pull/123. Typically inside a CI environment variable.")
	err := set.Parse(strArgs)
	if err != nil {
		return Args{}, err
	}

	set.VisitAll(func(f *flag.Flag) {
		if err == nil && strings.HasPrefix(f.Usage, "Required.") && f.Value.String() == "" {
			err = fmt.Errorf("flag -%s is required", f.Name)
		}
	})
	if err != nil {
		set.Usage()
		return Args{}, err
	}

	osFS := os.NewFS()
	if args.DiffFile != "-" {
		args.DiffFile = toFSPathSetErr(osFS, args.DiffFile, &err)
	}
	args.DiffBaseDir = toFSPathSetErr(osFS, args.DiffBaseDir, &err)
	args.GoCoverageFile = toFSPathSetErr(osFS, args.GoCoverageFile, &err)
	return args, err
}

func toFSPathSetErr(fs *os.FS, p string, err *error) string {
	p, pathErr := toFSPath(fs, p)
	setErr(pathErr, err)
	return p
}

func setErr(err error, setErr *error) {
	if err != nil && *setErr == nil {
		*setErr = err
	}
}

func toFSPath(fs *os.FS, p string) (string, error) {
	p, err := filepath.Abs(p)
	if err == nil {
		p, err = fs.FromOSPath(p)
	}
	return p, err
}

// Deps contains dependencies to inject into a covet run. Swapped out in tests.
type Deps struct {
	Stdin  io.Reader
	Stdout io.Writer
	FS     hackpadfs.FS
}

func runArgs(args Args, deps Deps) (err error) {
	defer func() { err = errors.WithStack(err) }()

	var diffFile io.Reader
	if args.DiffFile == "-" {
		diffFile = deps.Stdin
	} else {
		f, err := deps.FS.Open(args.DiffFile)
		if err != nil {
			return err
		}
		defer f.Close()
		diffFile = f
	}

	covet, err := covet.Parse(covet.Options{
		FS:             deps.FS,
		Diff:           diffFile,
		DiffBaseDir:    args.DiffBaseDir,
		GoCoveragePath: args.GoCoverageFile,
	})
	if err != nil {
		return err
	}
	if len(covet.DiffCoverageFiles()) == 0 {
		fmt.Fprintln(deps.Stdout, "No coverage information intersects with diff.")
		return nil
	}

	totalCovered := covet.DiffCovered()

	uncoveredFiles := findReportableUncoveredFiles(covet.DiffCoverageFiles(), float64(args.TargetDiffCoverage)/maxPercentInt, totalCovered)

	if args.ShowCoverage {
		for _, f := range uncoveredFiles {
			fmt.Fprintln(deps.Stdout, "Coverage diff:", f.Name)
			err := printCovet(deps.Stdout, deps.FS, f, args.GoCoverageFile)
			if err != nil {
				return err
			}
		}
	}

	fmt.Fprintln(deps.Stdout)
	totalCoveredStatus := coverstatus.New(totalCovered)
	fmt.Fprintln(deps.Stdout, "Total diff coverage:", totalCoveredStatus.Colorize(summary.FormatPercent(totalCovered)))
	fmt.Fprintln(deps.Stdout)
	summaryReport := summary.New(uncoveredFiles, args.TargetDiffCoverage, summary.FormatTable)
	fmt.Fprint(deps.Stdout, summaryReport)

	runWorkflow(coverageCommand(totalCovered, "", nil))
	for _, f := range uncoveredFiles {
		runWorkflow(coverageCommand(summary.FileCoverage(f), f.Name, findUncoveredLines(f)))
	}

	if args.GitHubToken != "" {
		org, repo, number, err := parseIssueURL(args.GitHubIssue)
		if err != nil {
			return err
		}
		err = ensureAppGitHubComment(context.Background(), gitHubCommentOptions{
			GitHubEndpoint: args.GitHubEndpoint,
			GitHubToken:    args.GitHubToken,
			RepoOwner:      org,
			Repo:           repo,
			IssueNumber:    number,
			Body:           summary.New(uncoveredFiles, args.TargetDiffCoverage, summary.FormatMarkdown),
		})
		if err != nil {
			fmt.Fprintln(deps.Stdout, "\nFailed to update GitHub comment, skipping. Error:", err)
		}
	}
	return nil
}

func printCovet(w io.Writer, fs hackpadfs.FS, f covet.File, covPath string) error {
	r, err := openFile(fs, f.Name, covPath)
	if err != nil {
		return err
	}
	defer r.Close()

	chunks, err := covet.DiffChunks(f, r)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		fmt.Fprintln(w, "Coverage:", chunk.FirstLine, "to", chunk.LastLine)
		for _, line := range chunk.Lines {
			switch {
			case strings.HasPrefix(line, "+"):
				line = color.GreenString(line)
			case strings.HasPrefix(line, "-"):
				line = color.RedString(line)
			}
			fmt.Fprintln(w, line)
		}
	}
	return nil
}

func openFile(fs hackpadfs.FS, name, covPath string) (io.ReadCloser, error) {
	name = path.Join(path.Dir(covPath), name)
	return fs.Open(name)
}

func findUncoveredLines(f covet.File) []span.Span {
	var uncoveredLines []span.Span
	ok := true
	var nextLineIndex int
	for ok {
		var uncovered span.Span
		uncovered, ok, nextLineIndex = findFirstUncoveredLines(f.Lines, nextLineIndex)
		if ok {
			uncoveredLines = append(uncoveredLines, uncovered)
		}
	}
	sort.SliceStable(uncoveredLines, func(a, b int) bool {
		return uncoveredLines[a].Len() > uncoveredLines[b].Len()
	})
	return uncoveredLines
}

func findFirstUncoveredLines(lines []covet.Line, startIndex int) (uncovered span.Span, ok bool, nextLineIndex int) {
	// find start
	nextLineIndex = startIndex
	for _, l := range lines[nextLineIndex:] {
		nextLineIndex++
		if !l.Covered {
			n := int64(l.LineNumber)
			uncovered = span.Span{
				Start: n,
				End:   n + 1,
			}
			ok = true
			break
		}
	}
	// find next line number jump or covered line
	for _, l := range lines[nextLineIndex:] {
		if l.Covered || int64(l.LineNumber) != uncovered.End {
			break
		}
		nextLineIndex++
		uncovered.End++
	}
	return
}

func findReportableUncoveredFiles(coveredFiles []covet.File, target, current float64) []covet.File {
	// sort by highest uncovered line count
	sort.Slice(coveredFiles, func(aIndex, bIndex int) bool {
		a, b := coveredFiles[aIndex], coveredFiles[bIndex]
		switch {
		case a.Uncovered != b.Uncovered:
			return a.Uncovered > b.Uncovered
		default:
			return a.Name < b.Name
		}
	})

	var uncoveredFiles []covet.File
	// find minimum number of covered lines required to hit target
	targetMissingLines := 0
	totalLines := uint(0)
	for _, f := range coveredFiles {
		totalLines += f.Covered + f.Uncovered
	}
	if percentDiff := target - current; percentDiff > 0 {
		targetMissingLines = int(percentDiff * float64(totalLines))
	} else {
		return nil // target is met
	}
	// next, collect the biggest uncovered files until we'd hit the target
	for _, f := range coveredFiles {
		const minUncoveredThreshold = 2 // include more files if it is slim pickings
		if f.Uncovered > 0 {
			uncoveredFiles = append(uncoveredFiles, f)
		}
		if f.Uncovered > minUncoveredThreshold {
			targetMissingLines -= int(f.Uncovered)
		}
		if targetMissingLines <= 0 {
			break
		}
	}
	return uncoveredFiles
}

const (
	decimalBase = 10
	maxIntBits  = 64
)

func parseIssueURL(s string) (org, repo string, number int, err error) {
	if s == "" {
		err = fmt.Errorf("-gh-issue is required")
		return
	}
	issueURL, err := url.Parse(s)
	if err != nil {
		return
	}
	if issueURL.Scheme == "" {
		issueURL, err = url.Parse("https://" + s)
		if err != nil {
			return
		}
	}
	const minIssueURLPathComponents = 4
	tokens := strings.SplitN(strings.TrimPrefix(issueURL.Path, "/"), "/", minIssueURLPathComponents+1)
	if len(tokens) < minIssueURLPathComponents {
		err = fmt.Errorf("malformed issue URL: expected 4+ path components, e.g. github.com/org/repo/pull/123")
		return
	}
	n, err := strconv.ParseInt(tokens[3], decimalBase, maxIntBits)
	if err != nil {
		return
	}
	org = tokens[0]
	repo = tokens[1]
	number = int(n)
	return
}
