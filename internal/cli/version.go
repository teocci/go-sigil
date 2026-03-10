package cli

import (
	"fmt"

	"go-sigil/internal/constants"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Sigil version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s\n", constants.AppName, constants.AppVersion)
		},
	}
}
