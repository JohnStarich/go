// Package summary generates summary reports in various formats.
package summary

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/johnstarich/go/covet/internal/coverfile"
	"github.com/johnstarich/go/covet/internal/coverstatus"
)

// New generates a new summary report in the given format
func New(uncoveredFiles []coverfile.File, targetCoverage uint, format Format) string {
	if len(uncoveredFiles) == 0 {
		return fmt.Sprintf("Successfully reached diff coverage target: >%d%%\n", targetCoverage)
	}

	var sb strings.Builder
	sb.WriteString("Diff coverage is below target. Add tests for these files:\n")
	tbl := table.NewWriter()
	const coverageColumnIndex = 2
	tbl.SetColumnConfigs([]table.ColumnConfig{
		{Number: coverageColumnIndex, Align: text.AlignCenter},
	})
	tbl.SuppressEmptyColumns()
	bold := boldColor()
	tbl.AppendHeader(table.Row{
		"",
		format.Colorize(bold, "Lines"),
		format.Colorize(bold, "Coverage"),
		format.Colorize(bold, "File"),
	})
	for _, f := range uncoveredFiles {
		percent := FileCoverage(f)
		status := coverstatus.New(percent)
		tbl.AppendRow(table.Row{
			format.StatusIcon(status),
			format.ColorizeStatus(status, format.Monospace(formatFraction(f.Covered, f.Uncovered+f.Covered))),
			format.ColorizeStatus(status, format.Monospace(FormatPercent(percent)+" "+formatGraph(percent, format))),
			f.Name,
		})
	}
	sb.WriteString(format.FormatTable(tbl))
	sb.WriteRune('\n')
	return sb.String()
}

// FileCoverage returns a File's coverage percentage between 0 and 1
func FileCoverage(f coverfile.File) float64 {
	return float64(f.Covered) / float64(f.Covered+f.Uncovered)
}
