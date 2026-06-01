package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeRepoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:repo <name>",
		Short: "Generate Postgres repository implementation",
		Long: `Generate the Postgres repository implementation for a domain.

Example:
  wapgo make:repo product`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeRepo(args[0])
		},
	}
}

func runMakeRepo(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	out := filepath.Join("internal", "repository", "postgres", n.Snake+"_repository.go")
	content, err := generator.DomainTemplateContent(generator.TemplateFS, "postgres_repository.go.tmpl")
	if err != nil {
		return err
	}
	if err := generator.Render(content, out, n); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)
	return nil
}
