package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMakeAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "make:all <name>",
		Short: "Generate all layers for a new domain at once",
		Long: `Generate every layer for a new domain in one command:

  make:model      → entity + domain repository interface
  make:repo       → Postgres repository implementation
  make:usecase    → usecase interface + implementation
  make:controller → HTTP handler + external service interface
  make:route      → route registration
  make:client     → inter-service HTTP client

Example:
  wapgo make:all product`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("Generating all layers for domain '%s'...\n\n", name)

			steps := []struct {
				label string
				fn    func(string) error
			}{
				{"model      (entity + repo interface)", runMakeModel},
				{"repo       (postgres impl)", runMakeRepo},
				{"usecase    (interface + impl)", runMakeUsecase},
				{"controller (handler + ext service)", runMakeController},
				{"route      (route registration)", runMakeRoute},
				{"client     (http client)", runMakeClient},
			}

			for _, s := range steps {
				fmt.Printf("[make:%s]\n", s.label)
				if err := s.fn(name); err != nil {
					return fmt.Errorf("make:%s failed: %w", s.label, err)
				}
				fmt.Println()
			}

			fmt.Printf("Domain '%s' generated. Don't forget to:\n", name)
			fmt.Printf("  1. Add fields to entity and DTOs\n")
			fmt.Printf("  2. Wire handler + repo in cmd/api/main.go\n")
			fmt.Printf("  3. Register route in internal/delivery/http/route/router.go\n")
			fmt.Printf("  4. Run: go build ./...\n")
			return nil
		},
	}
}
