package source

import (
	"net/url"
)

type LinkOptions struct {
	Line int
}

type Linker interface {
	LinkToSource(packagePath string, options LinkOptions) url.URL
}

type ScrapeChecker interface {
	ShouldScrapePackage(packagePath string) bool
}
