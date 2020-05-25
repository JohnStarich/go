package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/johnstarich/go/gopages/cmd"
	"github.com/johnstarich/go/gopages/internal/flags"
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

	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return genDoc(templateBytes, f)
}

func genDoc(templateBytes []byte, w io.Writer) error {
	tmpl := template.New("")
	tmpl.Funcs(funcMap())
	docTemplate, err := tmpl.Parse(string(templateBytes))
	if err != nil {
		return err
	}

	_, usageOutput, err := flags.Parse("-help")
	if err != flag.ErrHelp {
		return err
	}

	return docTemplate.Execute(w, map[string]interface{}{
		"Usage": usageOutput,
	})
}

func funcMap() template.FuncMap {
	return map[string]interface{}{
		"comment": func(s string) string {
			return strings.ReplaceAll(s, "\n", "\n// ")
		},
	}
}
