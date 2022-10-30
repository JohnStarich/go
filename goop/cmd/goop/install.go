package main

import (
	"github.com/urfave/cli/v2"
)

func (a App) install(c *cli.Context) error {
	pkg, err := a.parsePackagePattern(c.String("package"))
	if err != nil {
		return err
	}
	name := c.String("name")
	if name == "" {
		name = pkg.Name
	}
	if _, err := a.build(c.Context, name, pkg, true); err != nil {
		return err
	}
	return a.add(name, pkg)
}
