package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all domains generated in the current project",
		Long: `Scan internal/usecase/ for generated domain usecase files and print them.

Example:
  wapgo list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList()
		},
	}
}

func runList() error {
	pattern := filepath.Join("internal", "usecase", "*_usecase.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob: %w", err)
	}

	if len(matches) == 0 {
		fmt.Println("No generated domains found in internal/usecase/")
		return nil
	}

	domains := make([]string, 0, len(matches))
	for _, m := range matches {
		base := filepath.Base(m)
		domain := strings.TrimSuffix(base, "_usecase.go")
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	wd, _ := os.Getwd()
	fmt.Printf("Domains in %s:\n", wd)
	for _, d := range domains {
		fmt.Printf("  • %s\n", d)
	}
	return nil
}
