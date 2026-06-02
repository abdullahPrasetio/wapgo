package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags: -X 'github.com/abdullahPrasetio/wapgo/cli/commands.Version=x.y.z'
var Version = "dev"

// rootCmd is the base command for the wapgo CLI.
var rootCmd = &cobra.Command{
	Use:   "wapgo",
	Short: "wapgo — Web API Platform for Go",
	Long: `wapgo CLI provides two capabilities:

  wapgo new <project>    — scaffold a new wapgo project
  wapgo make:<layer> <name> — generate a domain layer inside an existing project

Run 'wapgo <command> --help' for more information.`,
}

// Execute adds all child commands to the root and sets flags,
// then runs the appropriate subcommand.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newNewCmd())
	rootCmd.AddCommand(newMakeModelCmd())
	rootCmd.AddCommand(newMakeRepoCmd())
	rootCmd.AddCommand(newMakeUsecaseCmd())
	rootCmd.AddCommand(newMakeControllerCmd())
	rootCmd.AddCommand(newMakeRouteCmd())
	rootCmd.AddCommand(newMakeClientCmd())
	rootCmd.AddCommand(newMakeAllCmd())
	rootCmd.AddCommand(newMakeMigrationCmd())
}
