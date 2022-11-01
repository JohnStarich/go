package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		code := 1
		var exitCoder cli.ExitCoder
		if errors.As(err, &exitCoder) {
			code = exitCoder.ExitCode()
		}
		os.Exit(code)
		return
	}
}
