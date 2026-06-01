package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the wapgo CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("wapgo version %s\n", Version)
		},
	}
}
