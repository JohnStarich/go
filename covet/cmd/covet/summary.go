package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/johnstarich/go/covet"
	"github.com/johnstarich/go/covet/internal/minmax"
)

type summaryFormat int

const (
	summaryTable summaryFormat = iota
	summaryMarkdown
)

func (f summaryFormat) Colorize(c *color.Color, s string) string {
	if f == summaryTable {
		return c.Sprint(s)
	}
	return s
}

func (f summaryFormat) ColorizeStatus(status coverageStatus, s string) string {
	if f == summaryTable {
		return status.Colorize(s)
	}
	return s
}

func (f summaryFormat) FormatTable(tbl table.Writer) string {
	switch f {
	case summaryMarkdown:
		return tbl.RenderMarkdown()
	default:
		tbl.SetStyle(table.StyleLight)
		return tbl.Render()
	}
}

func (f summaryFormat) Monospace(s string) string {
	const nonBreakingSpace = " " // ASCII 255, non-breaking space
	switch f {
	case summaryMarkdown:
		s = strings.ReplaceAll(s, " ", nonBreakingSpace)
		return fmt.Sprintf("``%s``", s)
	default:
		return s
	}
}

func (f summaryFormat) StatusIcon(status coverageStatus) string {
	switch f {
	case summaryMarkdown:
		return status.Emoji()
	default:
		return ""
	}
}

func covetSummary(uncoveredFiles []covet.File, targetCoverage uint, format summaryFormat) string {
	if len(uncoveredFiles) == 0 {
		return fmt.Sprintf("Successfully reached diff coverage target: >%d%%\n", targetCoverage)
	}

	var sb strings.Builder
	sb.WriteString("Diff coverage is below target. Add tests for these files:\n")
	tbl := table.NewWriter()
	tbl.SetColumnConfigs([]table.ColumnConfig{
		{Number: 2, Align: text.AlignCenter},
	})
	tbl.SuppressEmptyColumns()
	tbl.AppendHeader(table.Row{
		"",
		format.Colorize(boldColor, "Lines"),
		format.Colorize(boldColor, "Coverage"),
		format.Colorize(boldColor, "File"),
	})
	for _, f := range uncoveredFiles {
		percent := coveredFile(f)
		status := newCoverageStatus(percent)
		tbl.AppendRow(table.Row{
			format.StatusIcon(status),
			format.ColorizeStatus(status, format.Monospace(formatFraction(f.Covered, f.Uncovered+f.Covered))),
			format.ColorizeStatus(status, format.Monospace(formatPercent(percent)+" "+formatGraph(percent, format))),
			f.Name,
		})
	}
	sb.WriteString(format.FormatTable(tbl))
	sb.WriteRune('\n')
	return sb.String()
}

func formatWidth(s string, width uint) string {
	return fmt.Sprintf(fmt.Sprintf("%%%ds", width), s)
}

func formatWidthLeft(s string, width uint) string {
	return fmt.Sprintf(fmt.Sprintf("%%-%ds", width), s)
}

func formatFraction(numerator, denominator uint) string {
	nStr := fmt.Sprintf("%d", numerator)
	dStr := fmt.Sprintf("%d", denominator)
	width := uint(minmax.MaxInt(len(nStr), len(dStr)))
	return fmt.Sprintf("%s/%s", formatWidth(nStr, width), formatWidthLeft(dStr, width))
}

func formatPercent(f float64) string {
	return fmt.Sprintf("%5.1f%%", 100*f)
}

func formatGraph(f float64, format summaryFormat) string {
	const (
		graphWidth    = 5
		graphTickSize = 1.0 / graphWidth
	)
	total := graphWidth
	var graph strings.Builder
	if f == 0 {
		graph.WriteRune(percentRune(0))
		total--
	}
	for f > 0 {
		graph.WriteRune(percentRune(f / graphTickSize))
		f -= graphTickSize
		total--
	}
	for total > 0 {
		graph.WriteRune(' ')
		total--
	}
	return format.Colorize(boldColor, graph.String())
}

func percentRune(f float64) rune {
	if f > 1 {
		f = 1
	}
	if f < 0 {
		f = 0
	}
	const (
		runeTicks = 7 // 8 increments - 1st base rune
		baseRune  = '█'
	)
	unicodeOffset := runeTicks - int(f*runeTicks)
	return rune(baseRune + unicodeOffset)
}
