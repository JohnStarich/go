// Package source defines interfaces for generating URL links in documentation.
package source

import (
	"net/url"
)

// LinkOptions contains options for configuring hyperlinks to source files
type LinkOptions struct {
	Line int
}

// Linker can generate hyperlinks for any given package's file
type Linker interface {
	LinkToSource(packagePath string, options LinkOptions) url.URL
}

// ScrapeChecker returns whether or not a given package's path should be scraped and included in the output
type ScrapeChecker interface {
	ShouldScrapePackage(packagePath string) bool
}
