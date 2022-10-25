package main

import (
	"github.com/urfave/cli/v2"
)

func (a App) install(c *cli.Context) error {
	module, err := a.parseModulePathArg(c.String("module"))
	if err != nil {
		return err
	}
	if _, err := a.build(c.Context, module); err != nil {
		return err
	}
	return a.add(module)
}
