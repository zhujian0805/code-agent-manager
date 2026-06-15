package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Display current version",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), a.version)
			return err
		},
	}
}
