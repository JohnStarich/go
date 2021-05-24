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
// {{.Usage | wordWrap 80 | comment}}
//{{/* Do not remove the blank line below, otherwise this template is incorrectly displayed for the cmd/gendoc package. */}}

package main
