package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeRouteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:route <name>",
		Short: "Generate route registration for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeRoute(args[0])
		},
	}
}

func runMakeRoute(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	out := filepath.Join("internal", "delivery", "http", "route", n.Snake+"_route.go")
	content, err := generator.DomainTemplateContent(generator.TemplateFS, "route.go.tmpl")
	if err != nil {
		return err
	}
	if err := generator.Render(content, out, n); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)
	return nil
}
