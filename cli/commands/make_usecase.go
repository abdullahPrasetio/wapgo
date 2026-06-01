package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeUsecaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:usecase <name>",
		Short: "Generate usecase interface and implementation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeUsecase(args[0])
		},
	}
}

func runMakeUsecase(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	out := filepath.Join("internal", "usecase", n.Snake+"_usecase.go")
	content, err := generator.DomainTemplateContent(generator.TemplateFS, "usecase.go.tmpl")
	if err != nil {
		return err
	}
	if err := generator.Render(content, out, n); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)
	return nil
}
