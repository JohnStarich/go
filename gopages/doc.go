// gopages generates static files for Go documentation, formatted with godoc.
//
// Installation:
//   go get github.com/johnstarich/go/gopages
//
// Generate documentation for your module by running without any flags.
//
// A 'go.mod' file must be present in the current directory.
//   cd ./mymodule
//   gopages
//
// Usage of gopages:
//   -base string
//     	Base URL to use for static assets
//   -brand-description string
//     	Branding description in the top left of documentation
//   -brand-title string
//     	Branding title in the top left of documentation
//   -gh-pages
//     	Automatically commit the output path to the gh-pages branch. The current branch
//     	must be clean.
//   -gh-pages-token string
//     	The Git token to push with. Usually this is an API key.
//   -gh-pages-user string
//     	The Git username to push with
//   -out string
//     	Output path for static files (default "dist")
//   -source-link string
//     	Custom source code link template. Disables built-in source code pages. For
//     	example, "https://github.com/johnstarich/go/blob/master/gopages/{{.Path}}{{if .Line}}#L{{.Line}}{{end}}"
//     	generates links compatible with GitHub and GitLab. Must be a valid Go template
//     	and must generate valid URLs.
// 
//
package main
