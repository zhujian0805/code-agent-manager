package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

// extensionsCommand provides the `cam extension(s)` group.  Most subcommands
// are thin passthroughs to `gemini extensions <verb>` since the Python
// implementation does the same.  `browse` is a native implementation that
// queries geminicli.com/extensions.json so it works even when Gemini isn't
// installed.
func (a *App) extensionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "extension",
		Aliases: []string{"extensions", "ext"},
		Short:   "Manage AI assistant extensions (Gemini today)",
	}
	cmd.AddCommand(extensionBrowseCommand())
	for _, sub := range []struct {
		name, short string
	}{
		{"install", "Install an extension"},
		{"uninstall", "Uninstall an extension"},
		{"list", "List installed extensions"},
		{"update", "Update extensions"},
		{"disable", "Disable an extension"},
		{"enable", "Enable an extension"},
		{"link", "Link a local extension"},
		{"new", "Create a new extension"},
		{"validate", "Validate an extension"},
		{"settings", "Manage extension settings"},
	} {
		sub := sub
		cmd.AddCommand(&cobra.Command{
			Use:                sub.name + " [args...]",
			Short:              sub.short,
			DisableFlagParsing: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return passthroughGemini(append([]string{"extensions", sub.name}, args...))
			},
		})
	}
	return cmd
}

// extensionBrowseURL is the canonical extension catalog endpoint.  Tests can
// override it via the CAM_EXTENSION_BROWSE_URL env var so they don't hit the
// real network.
const extensionBrowseURL = "https://geminicli.com/extensions.json"

func extensionBrowseCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "browse",
		Short: "Browse available Gemini extensions from geminicli.com",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			url := extensionBrowseURL
			if override := os.Getenv("CAM_EXTENSION_BROWSE_URL"); override != "" {
				url = override
			}
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Get(url)
			if err != nil {
				return fmt.Errorf("browse: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("browse: HTTP %d", resp.StatusCode)
			}
			var payload any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				return fmt.Errorf("browse: decode: %w", err)
			}
			items := normalizeExtensions(payload)
			if len(items) == 0 {
				fmt.Fprintln(out, "No extensions found.")
				return nil
			}
			fmt.Fprintf(out, "Available Gemini Extensions (%d):\n\n", len(items))
			for i, ext := range items {
				name, _ := ext["extensionName"].(string)
				desc, _ := ext["extensionDescription"].(string)
				if desc == "" {
					desc, _ = ext["repoDescription"].(string)
				}
				full, _ := ext["fullName"].(string)
				url, _ := ext["url"].(string)
				stars := 0
				if s, ok := ext["stars"].(float64); ok {
					stars = int(s)
				}
				fmt.Fprintf(out, "%3d. %s\n    %s\n    by %s   ★ %d\n    %s\n\n",
					i+1, name, desc, full, stars, url)
			}
			return nil
		},
	}
}

func normalizeExtensions(payload any) []map[string]any {
	switch v := payload.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		if exts, ok := v["extensions"].([]any); ok {
			out := make([]map[string]any, 0, len(exts))
			for _, item := range exts {
				if m, ok := item.(map[string]any); ok {
					out = append(out, m)
				}
			}
			return out
		}
	}
	return nil
}

func passthroughGemini(args []string) error {
	bin, err := exec.LookPath("gemini")
	if err != nil {
		return errors.New("gemini CLI is required for extension operations (install with `cam install gemini-cli`)")
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	return nil
}
