package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/johnstarich/go/diffcover"
	"github.com/johnstarich/go/diffcover/internal/span"
)

func init() {
	if os.Getenv("CI") == "true" {
		color.NoColor = false
	}
}

var (
	boldColor = color.New(color.Bold)
)

type Args struct {
	DiffFile           string
	GoCoverageFile     string
	ShowCoverage       bool
	TargetDiffCoverage uint

	GitHubToken    string
	GitHubIssue    string
	GitHubEndpoint string
}

func main() {
	err := run(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
}

func run(strArgs []string) error {
	var args Args
	set := flag.NewFlagSet("diffcover", flag.ContinueOnError)
	set.StringVar(&args.DiffFile, "diff-file", "", "Required. Path to a diff file. Use '-' for stdin.")
	set.StringVar(&args.GoCoverageFile, "cover-go", "", "Required. Path to a Go coverage profile.")
	set.BoolVar(&args.ShowCoverage, "show-coverage", false, "Show the coverage diff in addition to the summary.")
	set.UintVar(&args.TargetDiffCoverage, "target-diff-coverage", 90, "Target total test coverage of new lines. Reports the biggest gaps needed to reach the target. Any number between 0 and 100.")
	set.StringVar(&args.GitHubToken, "gh-token", "", "GitHub access token to post and update a PR comment. If running in GitHub Actions, a comment may not be necessary.")
	set.StringVar(&args.GitHubEndpoint, "gh-api", "https://api.github.com", "GitHub API endpoint. Required for GitHub Enterprise.")
	set.StringVar(&args.GitHubIssue, "gh-issue", "", "GitHub issue or pull request URL. Example: github.com/org/repo/pull/123. Typically inside a CI environment variable.")
	err := set.Parse(strArgs)
	if err == flag.ErrHelp {
		return nil
	}
	if err != nil {
		return err
	}

	set.VisitAll(func(f *flag.Flag) {
		if err == nil && strings.HasPrefix(f.Usage, "Required.") && f.Value.String() == "" {
			err = fmt.Errorf("flag -%s is required", f.Name)
		}
	})
	if err != nil {
		return err
	}

	var diffFile io.Reader
	if args.DiffFile == "-" {
		diffFile = os.Stdin
	} else {
		f, err := os.Open(args.DiffFile)
		if err != nil {
			return err
		}
		defer f.Close()
		diffFile = f
	}

	coverageFile, err := os.Open(args.GoCoverageFile)
	if err != nil {
		return err
	}
	defer coverageFile.Close()

	diffcov, err := diffcover.Parse(diffcover.Options{
		Diff:       diffFile,
		GoCoverage: coverageFile,
	})
	if err != nil {
		return err
	}
	totalCovered := diffcov.Covered()

	uncoveredFiles := findReportableUncoveredFiles(diffcov.Files(), float64(args.TargetDiffCoverage)/100, totalCovered)

	if args.ShowCoverage {
		for _, f := range uncoveredFiles {
			fmt.Println("Coverage diff:", f.Name)
			err := printDiffCover(f, args.GoCoverageFile)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println()
	totalCoveredStatus := newCoverageStatus(totalCovered)
	fmt.Println("Total diff coverage:", totalCoveredStatus.Colorize(formatPercent(totalCovered)))
	fmt.Println()
	summary := diffcoverSummary(uncoveredFiles, args.TargetDiffCoverage, summaryTable)
	fmt.Print(summary)

	runWorkflow(coverageCommand(totalCovered, "", nil))
	for _, f := range uncoveredFiles {
		runWorkflow(coverageCommand(coveredFile(f), f.Name, findUncoveredLines(f)))
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
			Body:           diffcoverSummary(uncoveredFiles, args.TargetDiffCoverage, summaryMarkdown),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to update GitHub comment, skipping. Error:", err)
		}
	}
	return nil
}

func printDiffCover(f diffcover.File, covPath string) error {
	r, err := openFile(f.Name, covPath)
	if err != nil {
		return err
	}
	defer r.Close()

	chunks, err := diffcover.DiffChunks(f, r)
	if err != nil {
		return err
	}
	for _, chunk := range chunks {
		fmt.Println("Coverage:", chunk.FirstLine, "to", chunk.LastLine)
		for _, line := range chunk.Lines {
			switch {
			case strings.HasPrefix(line, "+"):
				line = color.GreenString(line)
			case strings.HasPrefix(line, "-"):
				line = color.RedString(line)
			}
			fmt.Println(line)
		}
	}
	return nil
}

func coveredFile(f diffcover.File) float64 {
	return float64(f.Covered) / float64(f.Covered+f.Uncovered)
}

func openFile(name, covPath string) (io.ReadCloser, error) {
	name = filepath.Join(filepath.Dir(covPath), name)
	return os.Open(name)
}

func findUncoveredLines(f diffcover.File) []span.Span {
	var uncoveredLines []span.Span
	var uncovered span.Span
	for uncovered.End < int64(len(f.Lines)) {
		uncovered = findFirstUncoveredLines(f.Lines[uncovered.End:])
		uncoveredLines = append(uncoveredLines, uncovered)
	}
	sort.Slice(uncoveredLines, func(a, b int) bool {
		return uncoveredLines[a].Len() > uncoveredLines[b].Len()
	})
	return uncoveredLines
}

func findFirstUncoveredLines(lines []diffcover.Line) span.Span {
	var uncovered span.Span
	for _, l := range lines {
		if uncovered.Start == 0 && !l.Covered {
			uncovered.Start = int64(l.LineNumber)
		} else if uncovered.Start != 0 && l.Covered {
			uncovered.End = int64(l.LineNumber + 1)
			break
		}
	}
	if uncovered.Start == 0 {
		uncovered.Start = int64(lines[0].LineNumber)
	}
	if uncovered.End == 0 {
		uncovered.End = int64(lines[len(lines)-1].LineNumber + 1)
	}
	return uncovered
}

func clampPercent(n uint) uint {
	if n > 100 {
		n = 100
	}
	return n
}

func findReportableUncoveredFiles(coveredFiles []diffcover.File, target, current float64) []diffcover.File {
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

	var uncoveredFiles []diffcover.File
	// find minimum number of covered lines required to hit target
	targetMissingLines := 0
	totalLines := uint(0)
	for _, f := range coveredFiles {
		totalLines += uint(f.Covered + f.Uncovered)
	}
	if percentDiff := target - current; percentDiff > 0 {
		targetMissingLines = int(percentDiff * float64(totalLines))
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
	tokens := strings.SplitN(strings.TrimPrefix(issueURL.Path, "/"), "/", 5)
	if len(tokens) < 4 {
		err = fmt.Errorf("malformed issue URL: expected 4+ path components, e.g. github.com/org/repo/pull/123")
		return
	}
	n, err := strconv.ParseInt(tokens[3], 10, 64)
	if err != nil {
		return
	}
	org = tokens[0]
	repo = tokens[1]
	number = int(n)
	return
}
