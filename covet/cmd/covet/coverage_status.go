package main

import "github.com/fatih/color"

type coverageStatus int

const (
	coverageExcellent coverageStatus = iota
	coverageGood
	coverageOK
	coverageWarning
	coverageError
)

func newCoverageStatus(f float64) coverageStatus {
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

func (c coverageStatus) WorkflowCommand() string {
	switch c {
	case coverageExcellent, coverageGood:
		return "notice"
	case coverageOK, coverageWarning:
		return "warning"
	default:
		return "error"
	}
}

var (
	boldGreen = color.New(color.Bold, color.FgGreen)
	boldRed   = color.New(color.Bold, color.FgRed)
)

func (c coverageStatus) Colorize(s string) string {
	switch c {
	case coverageExcellent:
		return boldGreen.Sprint(s)
	case coverageGood:
		return color.GreenString(s)
	case coverageOK:
		return color.YellowString(s)
	case coverageWarning:
		return color.RedString(s)
	default:
		return boldRed.Sprint(s)
	}
}

func (c coverageStatus) Emoji() string {
	switch c {
	case coverageExcellent, coverageGood:
		return "🟢"
	case coverageOK:
		return "🟡"
	case coverageWarning:
		return "🟠"
	default:
		return "🔴"
	}
}
