package summary

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/johnstarich/go/covet/internal/coverstatus"
	"github.com/johnstarich/go/covet/internal/minmax"
)

// Format represents a summary report's format
type Format int

// Supported formats
const (
	FormatColorTerminal Format = iota
	FormatMarkdown
)

// Colorize returns 's' and optionally wraps with color 'c' according to the format's rules
func (f Format) Colorize(c *color.Color, s string) string {
	if f == FormatColorTerminal {
		return c.Sprint(s)
	}
	return s
}

// ColorizeStatus returns 's' and optionally wraps with the coverage status's color according to the format's rules
func (f Format) ColorizeStatus(status coverstatus.Status, s string) string {
	if f == FormatColorTerminal {
		return status.Colorize(s)
	}
	return s
}

// FormatTable returns a formatted table according to the format's rules
func (f Format) FormatTable(tbl table.Writer) string {
	if f == FormatMarkdown {
		return tbl.RenderMarkdown()
	}
	tbl.SetStyle(table.StyleLight)
	return tbl.Render()
}

// Monospace returns a monospaced string according to the format's rules.
// Markdown uses code blocks with non-breaking spaces to ensure proper spacing.
func (f Format) Monospace(s string) string {
	const nonBreakingSpace = " " // ASCII 255, non-breaking space
	if f == FormatMarkdown {
		s = strings.ReplaceAll(s, " ", nonBreakingSpace)
		return fmt.Sprintf("``%s``", s)
	}
	return s
}

// StatusIcon returns a coverage status icon according to the format's rules.
// Markdown returns an emoji, terminal returns empty string.
func (f Format) StatusIcon(status coverstatus.Status) string {
	if f == FormatMarkdown {
		return status.Emoji()
	}
	return ""
}

func formatWidth(s string, width int) string {
	return fmt.Sprintf(fmt.Sprintf("%%%ds", width), s)
}

func formatWidthLeft(s string, width int) string {
	return fmt.Sprintf(fmt.Sprintf("%%-%ds", width), s)
}

func formatFraction(numerator, denominator uint) string {
	nStr := fmt.Sprintf("%d", numerator)
	dStr := fmt.Sprintf("%d", denominator)
	width := minmax.Max(len(nStr), len(dStr))
	return fmt.Sprintf("%s/%s", formatWidth(nStr, width), formatWidthLeft(dStr, width))
}

// FormatPercent formats a percentage of 0 to 1 with ABC.X% magnitude and precision
func FormatPercent(f float64) string {
	const maxPercentInt = 100
	return fmt.Sprintf("%5.1f%%", maxPercentInt*f)
}

func formatGraph(f float64, format Format) string {
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
	return format.Colorize(boldColor(), graph.String())
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

func boldColor() *color.Color { return color.New(color.Bold) }
