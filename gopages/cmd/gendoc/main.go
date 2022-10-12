package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/therve/go/gopages/cmd"
	"github.com/therve/go/gopages/internal/flags"
	"github.com/pkg/errors"
)

func main() {
	templatePath := flag.String("template", "", "Path to the desired doc template file")
	outPath := flag.String("out", "", "Output path of completed template")
	flag.Parse()
	if err := run(*templatePath, *outPath); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrap(err, "gendoc").Error())
		cmd.Exit(1)
	}
}

func run(templatePath, outPath string) error {
	if templatePath == "" || outPath == "" {
		return errors.New("Provide doc template and output file paths")
	}
	templateBytes, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}

	const genDocPerm = 0644
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, genDocPerm)
	if err != nil {
		return err
	}
	defer f.Close()
	return genDoc(templateBytes, f)
}

func genDoc(templateBytes []byte, w io.Writer) error {
	templateStr := string(templateBytes)
	templateStr = strings.Replace(templateStr, "\n\npackage main", "\npackage main", 1) // enable the comment for godoc by removing the blank line above 'package main'

	tmpl := template.New("")
	tmpl.Funcs(funcMap())
	docTemplate, err := tmpl.Parse(templateStr)
	if err != nil {
		return err
	}

	_, usageOutput, err := flags.Parse("-help")
	if !errors.Is(err, flag.ErrHelp) {
		return err
	}

	return docTemplate.Execute(w, map[string]interface{}{
		"Usage": usageOutput,
	})
}

func funcMap() template.FuncMap {
	return map[string]interface{}{
		"comment": func(s string) string {
			return strings.TrimSpace(strings.ReplaceAll(s, "\n", "\n// "))
		},
		"wordWrap": wordWrapLines,
	}
}
