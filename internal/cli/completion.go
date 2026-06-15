package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func (a *App) completionCommand(root *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "completion [bash|zsh|fish|powershell|pwsh]",
		Aliases: []string{"comp", "c"},
		Short:   "Generate shell completion scripts",
		Long: "Generate shell completion scripts using Cobra's native generator. " +
			"Source the output for the matching shell to enable command, flag, and value completions.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := strings.ToLower(args[0])
			normalized := normalizeShell(shell)
			if normalized == "" {
				return fmt.Errorf("Unsupported shell: %s", shell)
			}
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(out, "# code-agent-manager %s completion\n", normalized); err != nil {
				return err
			}
			switch normalized {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			}
			return fmt.Errorf("Unsupported shell: %s", shell)
		},
	}
	return cmd
}

func normalizeShell(shell string) string {
	switch shell {
	case "bash", "zsh", "fish", "powershell":
		return shell
	case "pwsh":
		return "powershell"
	}
	return ""
}
