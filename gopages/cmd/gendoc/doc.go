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
// NOTE: Install gopages with Go v1.19 or higher to generate documentation with custom links, lists, headings, and more.
// See: https://pkg.go.dev/go/doc/comment
//
// {{.Usage | wordWrap 80 | comment}}
// {{- /* Do not remove the blank line below, otherwise this template is incorrectly displayed for the cmd/gendoc package. */}}

package main
