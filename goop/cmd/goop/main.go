package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

var osExiter = os.Exit

func main() {
	err := run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExiter(exitCode(err))
		return
	}
}

func exitCode(err error) int {
	code := 1
	var exitCoder cli.ExitCoder
	if errors.As(err, &exitCoder) {
		code = exitCoder.ExitCode()
	}
	return code
}
