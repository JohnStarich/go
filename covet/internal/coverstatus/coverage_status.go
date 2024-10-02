// Package coverstatus contains coverage color-coding and labeling for generating reports.
package coverstatus

import "github.com/fatih/color"

// Status represents a coverage status level, ranging from "excellent" to "error"
type Status int

const (
	coverageExcellent Status = iota
	coverageGood
	coverageOK
	coverageWarning
	coverageError
)

// New categorizes the given percentage (between 0 and 1) as a coverage status
func New(f float64) Status {
	//nolint:mnd // These magic numbers are indeed arbitrary thresholds. As long as they are monotonically increasing from 0 to 1, we're ok.
	switch {
	case f < 0.50:
		return coverageError
	case f < 0.70:
		return coverageWarning
	case f < 0.80:
		return coverageOK
	case f < 0.90:
		return coverageGood
	default:
		return coverageExcellent
	}
}

// WorkflowCommand returns a GitHub Actions workflow command for this coverage status.
// These are specifically the log "level" commands.
func (s Status) WorkflowCommand() string {
	switch s {
	case coverageExcellent, coverageGood:
		return "notice"
	case coverageOK, coverageWarning:
		return "warning"
	case coverageError:
		return "error"
	default:
		return "error"
	}
}

func boldGreen() *color.Color { return color.New(color.Bold, color.FgGreen) }
func green() *color.Color     { return color.New(color.FgGreen) }
func yellow() *color.Color    { return color.New(color.FgYellow) }
func red() *color.Color       { return color.New(color.FgRed) }
func boldRed() *color.Color   { return color.New(color.Bold, color.FgRed) }

// Colorize formats 'str' with this status's assigned color
func (s Status) Colorize(str string) string {
	return s.color().Sprint(str)
}

func (s Status) color() *color.Color {
	switch s {
	case coverageExcellent:
		return boldGreen()
	case coverageGood:
		return green()
	case coverageOK:
		return yellow()
	case coverageWarning:
		return red()
	case coverageError:
		return boldRed()
	default:
		return boldRed()
	}
}

// Emoji returns this status's assigned emoji
func (s Status) Emoji() string {
	switch s {
	case coverageExcellent, coverageGood:
		return "🟢"
	case coverageOK:
		return "🟡"
	case coverageWarning:
		return "🟠"
	case coverageError:
		return "🔴"
	default:
		return "🔴"
	}
}
