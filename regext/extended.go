// Package regext enables extended regex, where spaces are ignored and inline comments are supported.
package regext

import (
	"regexp"
	"strings"
)

var (
	whitespaceRe = regexp.MustCompile(`\s+`)
	commentsRe   = regexp.MustCompile(`(?:\\#|[^#])*`)
)

// MustCompile compiles expr as a regular expression in a Perl-like "extended" mode where whitespace characters and comments are ignored
func MustCompile(expr string) *regexp.Regexp {
	return regexp.MustCompile(extendedRegexp(expr))
}

// Compile compiles expr as a regular expression in a Perl-like "extended" mode where whitespace characters and comments are ignored
func Compile(expr string) (*regexp.Regexp, error) {
	return regexp.Compile(extendedRegexp(expr))
}

func extendedRegexp(expr string) string {
	expr = removeComments(expr)
	expr = removeWhitespace(expr)
	return expr
}

func removeComments(expr string) string {
	lines := strings.Split(expr, "\n")
	for i := range lines {
		lines[i] = commentsRe.FindStringSubmatch(lines[i])[0]
	}
	return strings.Join(lines, "\n")
}

func removeWhitespace(expr string) string {
	return strings.Join(whitespaceRe.Split(expr, -1), "")
}
