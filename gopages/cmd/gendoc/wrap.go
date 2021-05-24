package main

import (
	"strings"
	"unicode"
)

// wordWrapLines breaks 's' into lines, then wraps each line's non-space chars to a max of 'columns'.
// Leading indentation is maintained on any new wrapped lines.
// The total column count may exceed 'columns' if there is leading indentation.
func wordWrapLines(columns int, s string) string {
	var sb strings.Builder
	lines := strings.SplitAfter(s, "\n")
	for _, line := range lines {
		sb.WriteString(wordWrapLine(columns, line))
	}
	return sb.String()
}

func nonSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

func wordWrapLine(columns int, line string) string {
	var prefix, suffix string
	firstNonSpace := strings.IndexFunc(line, nonSpace)
	lastNonSpace := strings.LastIndexFunc(line, nonSpace)
	switch {
	case firstNonSpace == -1 && lastNonSpace == -1:
		return line // only space chars found, return immediately
	case firstNonSpace != -1 && lastNonSpace != -1:
		prefix, suffix = line[:firstNonSpace], line[lastNonSpace+1:]
		line = line[firstNonSpace : lastNonSpace+1]
	default:
		panic("Impossible case. Either no non-space runes are found or none are found.")
	}

	lines := wordWrap(columns, line)
	return prefix + strings.Join(lines, "\n"+prefix) + suffix
}

func wordWrap(columns int, s string) []string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil
	}

	breakCandidates := []int{len(s)}
	if columns < len(s) {
		lastWordBreak := strings.LastIndexFunc(s[:columns], unicode.IsSpace)
		breakCandidates = append(breakCandidates, lastWordBreak)
	}
	avoidZones := nonBreakZones(s)
	breakCandidates = updateNonBreakZones(breakCandidates, avoidZones)
	nextLineBreak := smallestNonNegative(breakCandidates...)
	return append(
		[]string{s[:nextLineBreak]},
		wordWrap(columns, s[nextLineBreak:])...,
	)
}

type breakZone struct {
	min, max int
}

func (z breakZone) Contains(i int) bool {
	return z.min <= i && i < z.max
}

func (z breakZone) BestBreak() int {
	halfLength := (z.max - z.min) / 2
	if z.min > halfLength {
		return z.min
	}
	return z.max
}

// nonBreakZones returns index ranges where line breaks should be avoided
func nonBreakZones(s string) []breakZone {
	var nonBreakZones []breakZone
	lastQuote := -1
	for ix, r := range s {
		if r == '"' {
			if lastQuote == -1 {
				lastQuote = ix
			} else {
				nonBreakZones = append(nonBreakZones, breakZone{
					min: lastQuote,
					max: ix + 1,
				})
				lastQuote = -1
			}
		}
	}
	return nonBreakZones
}

func updateNonBreakZones(breakCandidates []int, avoidZones []breakZone) []int {
	var validCandidates []int
	for _, c := range breakCandidates {
		validIndex := c
		for _, zone := range avoidZones {
			if zone.Contains(c) {
				validIndex = zone.BestBreak()
				break
			}
		}
		validCandidates = append(validCandidates, validIndex)
	}
	return validCandidates
}

func smallestNonNegative(values ...int) (x int) {
	var result int
	firstPositiveIndex := -1
	for i, val := range values {
		if val >= 0 {
			firstPositiveIndex, result = i, val
			break
		}
	}
	for _, val := range values[firstPositiveIndex+1:] {
		if val >= 0 && val < result {
			result = val
		}
	}
	return result
}
