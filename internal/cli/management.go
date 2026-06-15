package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/chat2anyllm/code-agent-manager/internal/entities"
	"github.com/chat2anyllm/code-agent-manager/internal/fetching"
)

// managementCommand returns the prompt/skill/agent/plugin command tree, all
// backed by the unified entities package.  Group names are singular ("prompt",
// not "prompts") so help text matches Python.
func (a *App) managementCommand(group, alias string, state *globalState) *cobra.Command {
	kind := groupKind(group)
	cmd := &cobra.Command{
		Use:     group,
		Aliases: []string{alias},
		Short:   "Manage " + group + " configurations",
	}
	cmd.AddCommand(entityListCommand(kind))
	cmd.AddCommand(entityShowCommand(kind))
	cmd.AddCommand(entityAddCommand(kind))
	cmd.AddCommand(entityRemoveCommand(kind))
	cmd.AddCommand(entityInstallCommand(kind))
	cmd.AddCommand(entityUninstallCommand(kind))
	cmd.AddCommand(entityFetchCommand(kind))
	cmd.AddCommand(entityReposCommand(kind))
	cmd.AddCommand(entityExportCommand(kind))
	cmd.AddCommand(entityImportCommand(kind))
	cmd.AddCommand(entityInstalledCommand(kind))
	return cmd
}

func groupKind(group string) entities.Kind {
	switch group {
	case "prompt":
		return entities.KindPrompt
	case "skill":
		return entities.KindSkill
	case "agent":
		return entities.KindAgent
	case "plugin":
		return entities.KindPlugin
	}
	return entities.Kind(group)
}

func entityListCommand(kind entities.Kind) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed " + string(kind) + "s",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := entities.NewStore(kind)
			items, err := store.All()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(items) == 0 {
				fmt.Fprintf(out, "No %ss installed\n", kind)
				return nil
			}
			for _, e := range items {
				desc := e.Description
				if desc == "" {
					desc = e.Path
				}
				fmt.Fprintf(out, "%-40s %s\n", e.Name, desc)
			}
			return nil
		},
	}
}

func entityShowCommand(kind entities.Kind) *cobra.Command {
	return &cobra.Command{
		Use:     "show NAME",
		Aliases: []string{"view"},
		Short:   "Show " + string(kind) + " details",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := entities.NewStore(kind)
			e, err := store.Get(args[0])
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(e, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}

func entityAddCommand(kind entities.Kind) *cobra.Command {
	var (
		description string
		contentFile string
		tags        []string
	)
	cmd := &cobra.Command{
		Use:     "add NAME",
		Aliases: []string{"create"},
		Short:   "Add a " + string(kind),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := entities.NewStore(kind)
			entity := entities.Entity{Name: args[0], Description: description, Tags: tags}
			if contentFile != "" {
				data, err := os.ReadFile(contentFile)
				if err != nil {
					return err
				}
				entity.Content = string(data)
			} else {
				stdin, err := readStdinIfPiped(cmd.InOrStdin())
				if err != nil {
					return err
				}
				entity.Content = stdin
			}
			if err := store.Put(entity); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added %s %s\n", kind, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVarP(&contentFile, "file", "f", "", "Read content from file")
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "Tag (repeatable)")
	return cmd
}

func entityRemoveCommand(kind entities.Kind) *cobra.Command {
	return &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"delete", "rm"},
		Short:   "Remove a " + string(kind),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store := entities.NewStore(kind)
			removed, err := store.Delete(args[0])
			if err != nil {
				return err
			}
			if !removed {
				fmt.Fprintf(cmd.OutOrStdout(), "Not found: %s\n", args[0])
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", args[0])
			return nil
		},
	}
}

func entityInstallCommand(kind entities.Kind) *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "install NAME",
		Short: "Install a " + string(kind) + " into an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required (one of: %s)", strings.Join(entities.SupportedApps(kind), ", "))
			}
			store := entities.NewStore(kind)
			e, err := store.Get(args[0])
			if err != nil {
				return err
			}
			dest, err := entities.InstallToApp(e, kind, app)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed %s to %s (%s)\n", e.Name, dest, app)
			return nil
		},
	}
	cmd.Flags().StringVarP(&app, "app", "a", "", "Target app")
	return cmd
}

func entityUninstallCommand(kind entities.Kind) *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "uninstall NAME",
		Short: "Uninstall a " + string(kind) + " from an app",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required (one of: %s)", strings.Join(entities.SupportedApps(kind), ", "))
			}
			_, removed, err := entities.UninstallFromApp(args[0], kind, app)
			if err != nil {
				return err
			}
			if !removed {
				fmt.Fprintf(cmd.OutOrStdout(), "Not installed for %s: %s\n", app, args[0])
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Uninstalled %s from %s\n", args[0], app)
			return nil
		},
	}
	cmd.Flags().StringVarP(&app, "app", "a", "", "Target app")
	return cmd
}

func entityFetchCommand(kind entities.Kind) *cobra.Command {
	var (
		owner  string
		repo   string
		branch string
		path   string
	)
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch " + string(kind) + "s from a GitHub repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if owner == "" || repo == "" {
				return errors.New("--owner and --repo are required")
			}
			client := fetching.New()
			dest := filepath.Join(os.TempDir(), fmt.Sprintf("cam-fetch-%s-%s", owner, repo))
			_ = os.RemoveAll(dest)
			root, err := client.DownloadGitHubZip(owner, repo, branch, dest)
			if err != nil {
				return err
			}
			scanRoot := root
			if path != "" {
				scanRoot = filepath.Join(root, path)
			}
			store := entities.NewStore(kind)
			added := 0
			fileTarget := defaultManifestName(kind)
			err = filepath.WalkDir(scanRoot, func(p string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				if filepath.Base(p) != fileTarget {
					return nil
				}
				data, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				name := filepath.Base(filepath.Dir(p))
				entity := entities.Entity{
					Name:    name,
					Content: string(data),
					Path:    p,
					Repo:    &entities.RepoRef{Owner: owner, Name: repo, Branch: branch, Path: path},
				}
				if err := store.Put(entity); err != nil {
					return err
				}
				added++
				return nil
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Fetched %d %ss from %s/%s\n", added, kind, owner, repo)
			return nil
		},
	}
	cmd.Flags().StringVarP(&owner, "owner", "o", "", "GitHub owner")
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "GitHub repo")
	cmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch")
	cmd.Flags().StringVarP(&path, "path", "p", "", "Sub-directory within the repo")
	return cmd
}

func defaultManifestName(kind entities.Kind) string {
	switch kind {
	case entities.KindSkill:
		return "SKILL.md"
	case entities.KindAgent:
		return "AGENT.md"
	case entities.KindPlugin:
		return "plugin.json"
	}
	return "README.md"
}

func entityReposCommand(kind entities.Kind) *cobra.Command {
	return &cobra.Command{
		Use:   "repos",
		Short: "List configured " + string(kind) + " repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "Repository management for %s is done via ~/.config/code-agent-manager/%s_repos.json\n", kind, kind)
			return nil
		},
	}
}

func entityExportCommand(kind entities.Kind) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export " + string(kind) + "s to JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store := entities.NewStore(kind)
			items, err := store.All()
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(items, "", "  ")
			if file == "" {
				_, err := cmd.OutOrStdout().Write(data)
				return err
			}
			return os.WriteFile(file, data, 0o600)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Export destination (stdout if empty)")
	return cmd
}

func entityImportCommand(kind entities.Kind) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import " + string(kind) + "s from JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return errors.New("--file is required")
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			var imported []entities.Entity
			if err := json.Unmarshal(data, &imported); err != nil {
				return err
			}
			store := entities.NewStore(kind)
			for _, e := range imported {
				if err := store.Put(e); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d %ss\n", len(imported), kind)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON file to read")
	return cmd
}

func entityInstalledCommand(kind entities.Kind) *cobra.Command {
	return &cobra.Command{
		Use:   "installed",
		Short: "List installations across apps",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			any := false
			for _, app := range entities.SupportedApps(kind) {
				apps := entities.AppPathsFor(kind)
				dest := os.ExpandEnv(strings.ReplaceAll(apps[app], "~", os.Getenv("HOME")))
				if _, err := os.Stat(dest); err != nil {
					continue
				}
				entriesOnDisk, _ := os.ReadDir(dest)
				if len(entriesOnDisk) == 0 {
					continue
				}
				any = true
				fmt.Fprintf(out, "%s (%s):\n", app, dest)
				for _, e := range entriesOnDisk {
					fmt.Fprintf(out, "  %s\n", e.Name())
				}
			}
			if !any {
				fmt.Fprintf(out, "No installed %ss across apps\n", kind)
			}
			return nil
		},
	}
}

func readStdinIfPiped(in io.Reader) (string, error) {
	if file, ok := in.(*os.File); ok {
		info, err := file.Stat()
		if err != nil {
			return "", nil
		}
		if info.Mode()&os.ModeCharDevice != 0 {
			return "", nil
		}
		data, err := io.ReadAll(file)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
