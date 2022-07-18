package covet

import (
	"io"
	"sort"

	"github.com/johnstarich/go/covet/internal/summary"
)

// ReportSummaryOptions contains summary report options
type ReportSummaryOptions struct {
	Target uint
}

// ReportSummaryMarkdown writes a markdown report to 'w'.
// To capture as a string, write to a bytes.Buffer, then call buf.String().
func (c *Covet) ReportSummaryMarkdown(w io.Writer, options ReportSummaryOptions) error {
	return c.reportSummary(w, options, summary.FormatMarkdown)
}

// ReportSummaryColorTerminal writes a plain text report with color to 'w'.
func (c *Covet) ReportSummaryColorTerminal(w io.Writer, options ReportSummaryOptions) error {
	return c.reportSummary(w, options, summary.FormatColorTerminal)
}

func (c *Covet) reportSummary(w io.Writer, options ReportSummaryOptions, format summary.Format) error {
	uncoveredFiles := c.PriorityUncoveredFiles(options.Target)
	report := summary.New(uncoveredFiles, options.Target, format)
	_, err := io.WriteString(w, report)
	return err
}

// PriorityUncoveredFiles returns a list of Files prioritized by the largest uncovered sections of the diff.
// The list contains just enough Files worth of uncovered lines necessary to meet the provided 'target' coverage.
func (c *Covet) PriorityUncoveredFiles(target uint) []File {
	coveredFiles := c.DiffCoverageFiles()
	const maxPercentInt = 100
	targetPercent := float64(target) / maxPercentInt
	current := c.DiffCovered()
	return findReportableUncoveredFiles(coveredFiles, targetPercent, current)
}

func findReportableUncoveredFiles(coveredFiles []File, target, current float64) []File {
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

	var uncoveredFiles []File
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
