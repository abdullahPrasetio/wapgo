package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeClientCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:client <name>",
		Short: "Generate inter-service HTTP client for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeClient(args[0])
		},
	}
}

func runMakeClient(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	out := filepath.Join("pkg", "httpclient", n.Snake+"_client.go")
	content, err := generator.DomainTemplateContent(generator.TemplateFS, "http_client.go.tmpl")
	if err != nil {
		return err
	}
	if err := generator.Render(content, out, n); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)
	return nil
}
