// Command gopages generates static files for Go documentation, formatted with godoc.
//
// # Installation
//
// To install gopages, run the following command:
//
//	go install github.com/johnstarich/go/gopages@latest
//
// # Getting started
//
// Generate documentation for your module by running gopages without any flags.
//
// A 'go.mod' file must be present in the current directory.
//
//	cd ./mymodule
//	gopages
//
// NOTE: Install gopages with Go v1.19 or higher to generate documentation with [improved formatting].
//
// Usage of gopages:
//
//	-base string
//	  	Base URL to use for static assets
//	-brand-description string
//	  	Branding description in the top left of documentation
//	-brand-title string
//	  	Branding title in the top left of documentation
//	-gh-pages
//	  	Automatically commit the output path to the gh-pages branch. The current branch
//	  	must be clean.
//	-gh-pages-token string
//	  	The Git token to push with. Usually this is an API key.
//	-gh-pages-user string
//	  	The Git username to push with
//	-include-head value
//	  	Includes the given HTML file's contents in every page's '<head></head>'. Useful
//	  	for including custom analytics scripts. Must be valid HTML.
//	-internal
//	  	Includes 'internal' packages in the package index and unexported functions.
//	  	Useful for sharing documentation within the same development team. Note: This
//	  	only affects page generation for non-internal packages, like package lists.
//	  	Internal package docs are always generated.
//	-out string
//	  	Output path for static files (default "dist")
//	-source-link string
//	  	Custom source code link template. Disables built-in source code pages. For
//	  	example, "https://github.com/johnstarich/go/blob/master/gopages/{{.Path}}{{if .Line}}#L{{.Line}}{{end}}"
//	  	generates links compatible with GitHub and GitLab. Must be a valid Go template
//	  	and must generate valid URLs.
//
// [improved formatting]: https://pkg.go.dev/go/doc/comment
package main
