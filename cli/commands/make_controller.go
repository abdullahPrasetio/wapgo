package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeControllerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:controller <name>",
		Short: "Generate HTTP handler (controller) and external service interface",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeController(args[0])
		},
	}
}

func runMakeController(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	files := []struct {
		tmpl string
		out  string
	}{
		{"handler.go.tmpl", filepath.Join("internal", "delivery", "http", "handler", n.Snake+"_handler.go")},
		{"external_service.go.tmpl", filepath.Join("internal", "domain", "service", "external_"+n.Snake+".go")},
	}

	for _, f := range files {
		content, err := generator.DomainTemplateContent(generator.TemplateFS, f.tmpl)
		if err != nil {
			return err
		}
		if err := generator.Render(content, f.out, n); err != nil {
			return fmt.Errorf("generate %s: %w", f.out, err)
		}
		fmt.Printf("  created  %s\n", f.out)
	}
	return nil
}
