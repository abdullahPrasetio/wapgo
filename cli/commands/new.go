package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/abdullahPrasetio/wapgo/cli/generator"
)

func newNewCmd() *cobra.Command {
	var module string
	var db string

	cmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Scaffold a new wapgo project",
		Long: `Create a new wapgo project in a directory named <project-name>.

Examples:
  wapgo new my-service
  wapgo new shop --module github.com/me/shop --db mysql`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]

			if module == "" {
				module = "github.com/example/" + strings.ReplaceAll(projectName, "_", "-")
			}

			targetDir := filepath.Join(".", projectName)

			fmt.Printf("Scaffolding wapgo project '%s'...\n", projectName)
			fmt.Printf("  module : %s\n", module)
			fmt.Printf("  db     : %s\n", db)
			fmt.Printf("  target : %s\n\n", targetDir)

			opts := generator.ScaffoldOptions{
				ProjectName: projectName,
				Module:      module,
				DB:          db,
			}

			if err := generator.Scaffold(generator.TemplateFS, opts, targetDir); err != nil {
				return fmt.Errorf("scaffold failed: %w", err)
			}

			fmt.Println("Project created successfully!")
			fmt.Println()
			fmt.Printf("  cd %s\n", projectName)
			fmt.Printf("  cp .env.example .env\n")
			fmt.Printf("  make docker-up\n")
			fmt.Printf("  make run\n")
			fmt.Println()
			fmt.Println("Then add a new domain:")
			fmt.Printf("  wapgo make:all product\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&module, "module", "", "Go module path (default: github.com/example/<project-name>)")
	cmd.Flags().StringVar(&db, "db", "postgres", "Database driver: postgres | mysql")

	// Prevent accidental overwrite of current directory.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		projectName := args[0]
		if _, err := os.Stat(projectName); err == nil {
			return fmt.Errorf("directory '%s' already exists", projectName)
		}
		return nil
	}

	return cmd
}
