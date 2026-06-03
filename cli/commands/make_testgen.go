package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeTestCmd() *cobra.Command {
	var layer string

	cmd := &cobra.Command{
		Use:   "make:test <name>",
		Short: "Generate test boilerplate for a domain layer",
		Long: `Generate test boilerplate for a domain layer.

Layers:
  usecase  (default) — usecase unit tests with mock repository
  handler             — HTTP handler tests with mock usecase + httptest

Examples:
  wapgo make:test product
  wapgo make:test product --layer handler`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch layer {
			case "usecase", "":
				return runMakeTest(args[0])
			case "handler":
				return runMakeHandlerTest(args[0])
			default:
				return fmt.Errorf("unknown layer %q — valid: usecase, handler", layer)
			}
		},
	}

	cmd.Flags().StringVarP(&layer, "layer", "l", "usecase", "layer to generate test for (usecase|handler)")
	return cmd
}

func runMakeTest(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	out := filepath.Join("internal", "usecase", n.Snake+"_usecase_test.go")
	content, err := generator.DomainTemplateContent(generator.TemplateFS, "usecase_test.go.tmpl")
	if err != nil {
		return err
	}
	if err := generator.Render(content, out, n); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)
	return nil
}

func runMakeHandlerTest(name string) error {
	module, err := generator.ReadModulePath()
	if err != nil {
		return err
	}

	n := generator.NewNames(name)
	n.Module = module

	out := filepath.Join("internal", "delivery", "http", "handler", n.Snake+"_handler_test.go")
	content, err := generator.DomainTemplateContent(generator.TemplateFS, "handler_test.go.tmpl")
	if err != nil {
		return err
	}
	if err := generator.Render(content, out, n); err != nil {
		return fmt.Errorf("generate %s: %w", out, err)
	}
	fmt.Printf("  created  %s\n", out)
	return nil
}
