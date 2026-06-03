package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newMakeEventCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:event <name>",
		Short: "Generate Kafka event producer + consumer boilerplate for a domain",
		Long: `Generate a producer and consumer pair for Kafka domain events.

Files created:
  internal/event/{name}_producer.go  — typed event publisher
  internal/event/{name}_consumer.go  — typed event subscriber

Example:
  wapgo make:event order`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMakeEvent(args[0])
		},
	}
}

func runMakeEvent(name string) error {
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
		{"event_producer.go.tmpl", filepath.Join("internal", "event", n.Snake+"_producer.go")},
		{"event_consumer.go.tmpl", filepath.Join("internal", "event", n.Snake+"_consumer.go")},
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
