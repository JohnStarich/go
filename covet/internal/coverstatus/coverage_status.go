package coverstatus

import "github.com/fatih/color"

type Status int

const (
	coverageExcellent Status = iota
	coverageGood
	coverageOK
	coverageWarning
	coverageError
)

func New(f float64) Status {
	// nolint:gomnd // These magic numbers are indeed arbitrary thresholds. As long as they are monotonically increasing from 0 to 1, we're ok.
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
