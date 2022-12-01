package main

import "github.com/spf13/cobra"

func (a App) install(cmd *cobra.Command, args []string) error {
	pkgPattern, err := cmd.Flags().GetString("package")
	if err != nil {
		return err
	}
	pkg, err := a.parsePackagePattern(pkgPattern)
	if err != nil {
		return err
	}
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	if name == "" {
		name = pkg.Name
	}
	if _, err := a.build(cmd.Context(), name, pkg, true); err != nil {
		return err
	}
	return a.add(name, pkg)
}
