package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chat2anyllm/code-agent-manager/internal/mcp"
)

// mcpCommand wires `cam mcp` and `cam mcp server *` against the bundled
// MCP registry and per-client config writers.
func (a *App) mcpCommand(state *globalState) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "mcp",
		Aliases: []string{"m"},
		Short:   "Manage MCP servers",
	}
	cmd.AddCommand(a.mcpListCommand(state))
	cmd.AddCommand(a.mcpAddCommand(state))
	cmd.AddCommand(a.mcpRemoveCommand(state))
	cmd.AddCommand(a.mcpServerCommand())
	return cmd
}

// `cam mcp list` lists local installs grouped by client.  When --client is
// given, only that client is shown.
func (a *App) mcpListCommand(state *globalState) *cobra.Command {
	var clientName string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed MCP servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			clients := clientsForList(clientName)
			any := false
			for _, c := range clients {
				servers, _, err := mcp.ListServers(c, mcp.UserScope)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "  warning: %s: %v\n", c.Name, err)
					continue
				}
				if len(servers) == 0 {
					continue
				}
				any = true
				fmt.Fprintf(out, "%s:\n", c.Name)
				for _, s := range servers {
					fmt.Fprintf(out, "  %s\t%s\t%s\n", s.Name, s.Command, strings.Join(s.Args, " "))
				}
			}
			if !any {
				fmt.Fprintln(out, "No MCP servers installed")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&clientName, "client", "c", "", "Limit to a single client")
	return cmd
}

// `cam mcp add NAME [--client TOOL] [--command bin] [--arg ARG]...` installs
// either a registry server (when --command absent) or a one-off custom server.
func (a *App) mcpAddCommand(state *globalState) *cobra.Command {
	var (
		clientName string
		command    string
		serverArgs []string
		envEntries []string
		scope      string
		urlValue   string
	)
	cmd := &cobra.Command{
		Use:   "add NAME",
		Short: "Add MCP server to a client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			name := args[0]
			client, err := requireClient(clientName)
			if err != nil {
				return err
			}
			server := mcp.Server{Name: name, Command: command, Args: serverArgs, URL: urlValue, Env: parseEnv(envEntries)}
			if command == "" && urlValue == "" {
				registry, err := mcp.LoadBundledRegistry()
				if err != nil {
					return err
				}
				schema, ok := registry.Get(name)
				if !ok {
					return fmt.Errorf("server %q not found in registry (use --command for custom servers)", name)
				}
				server, err = mcp.ServerFromSchema(schema)
				if err != nil {
					return err
				}
			}
			if server.Type == "" {
				if server.URL != "" {
					server.Type = "http"
				} else {
					server.Type = "stdio"
				}
			}
			path, err := mcp.AddServer(client, mcp.Scope(scope), server)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Added %s to %s\n  Config: %s\n", name, client.Name, path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&clientName, "client", "c", "", "Target MCP client (required)")
	cmd.Flags().StringVar(&command, "command", "", "Server command (for custom STDIO servers)")
	cmd.Flags().StringArrayVar(&serverArgs, "arg", nil, "Server command argument (repeatable)")
	cmd.Flags().StringArrayVar(&envEntries, "env", nil, "Env var KEY=VALUE for the server (repeatable)")
	cmd.Flags().StringVarP(&scope, "scope", "s", "user", "Scope (user|project)")
	cmd.Flags().StringVar(&urlValue, "url", "", "Remote MCP server URL")
	_ = cmd.MarkFlagRequired("client")
	return cmd
}

// `cam mcp remove NAME --client TOOL`.
func (a *App) mcpRemoveCommand(state *globalState) *cobra.Command {
	var (
		clientName string
		scope      string
	)
	cmd := &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"delete", "rm"},
		Short:   "Remove MCP server from a client",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			client, err := requireClient(clientName)
			if err != nil {
				return err
			}
			path, found, err := mcp.RemoveServer(client, mcp.Scope(scope), args[0])
			if err != nil {
				return err
			}
			if !found {
				fmt.Fprintf(out, "Not installed: %s\n", args[0])
				return nil
			}
			fmt.Fprintf(out, "Removed %s from %s\n  Config: %s\n", args[0], client.Name, path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&clientName, "client", "c", "", "Target MCP client (required)")
	cmd.Flags().StringVarP(&scope, "scope", "s", "user", "Scope (user|project)")
	_ = cmd.MarkFlagRequired("client")
	return cmd
}

// `cam mcp server <list|search|show>` queries the bundled registry.
func (a *App) mcpServerCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "server", Short: "Browse the bundled MCP registry"}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List bundled servers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mcp.LoadBundledRegistry()
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
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "search QUERY",
		Short: "Search bundled servers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mcp.LoadBundledRegistry()
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
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show NAME",
		Short: "Show bundled server schema",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := mcp.LoadBundledRegistry()
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
	})
	return cmd
}

func clientsForList(name string) []mcp.ClientSpec {
	if name == "" {
		return mcp.SupportedClients
	}
	if c, ok := mcp.ClientByName(name); ok {
		return []mcp.ClientSpec{c}
	}
	return nil
}

func requireClient(name string) (mcp.ClientSpec, error) {
	if name == "" {
		return mcp.ClientSpec{}, fmt.Errorf("--client is required")
	}
	c, ok := mcp.ClientByName(name)
	if !ok {
		return mcp.ClientSpec{}, fmt.Errorf("unsupported client: %s (try one of: %s)", name, strings.Join(mcp.ClientNames(), ", "))
	}
	return c, nil
}

func parseEnv(entries []string) map[string]string {
	if len(entries) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, e := range entries {
		if i := strings.IndexByte(e, '='); i > 0 {
			out[e[:i]] = e[i+1:]
		}
	}
	return out
}
