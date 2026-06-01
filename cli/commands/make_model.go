package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeModelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:model <name>",
		Short: "Generate entity + domain repository interface",
		Long: `Generate the entity struct and domain repository interface for a new domain.

Example:
  wapgo make:model product`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeModel(args[0])
		},
	}
}

func runMakeModel(name string) error {
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
		{"entity.go.tmpl", filepath.Join("internal", "domain", "entity", n.Snake+".go")},
		{"repository_interface.go.tmpl", filepath.Join("internal", "domain", "repository", n.Snake+"_repository.go")},
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
