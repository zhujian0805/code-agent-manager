package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chat2anyllm/code-agent-manager/internal/mcp"
)

// `cam mcp server <list|search|show>` queries the catalog registry.
func (a *App) mcpServerCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "server", Short: "Browse the MCP catalog registry"}
	cmd.AddCommand(mcpServerListCommand())
	cmd.AddCommand(mcpServerSearchCommand())
	cmd.AddCommand(mcpServerShowCommand())
	return cmd
}

func mcpServerListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List catalog servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mcp.LoadRegistry()
			if err != nil {
				return err
			}
			for _, name := range r.Names() {
				s, _ := r.Get(name)
				display := s.DisplayName
				if display == "" {
					display = s.Name
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%-40s %s\n", s.Name, display)
			}
			return nil
		},
	}
}

func mcpServerSearchCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "search QUERY",
		Short: "Search catalog servers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mcp.LoadRegistry()
			if err != nil {
				return err
			}
			matches := r.Search(args[0])
			for _, s := range matches {
				fmt.Fprintf(cmd.OutOrStdout(), "%-40s %s\n", s.Name, s.Description)
			}
			if len(matches) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No matches for %q\n", args[0])
			}
			return nil
		},
	}
}

func mcpServerShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show catalog server schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mcp.LoadRegistry()
			if err != nil {
				return err
			}
			s, ok := r.Get(args[0])
			if !ok {
				return fmt.Errorf("server not found: %s", args[0])
			}
			b, _ := json.MarshalIndent(s, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}
}
