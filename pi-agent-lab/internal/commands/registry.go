package commands

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/darkokos/pi-agent-framework/pi-agent-lab/internal/format"
	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func NewRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Create, register, and list registries",
	}

	cmd.AddCommand(newRegistryInitCommand())
	cmd.AddCommand(newRegistryAddCommand())
	cmd.AddCommand(newRegistryListCommand())

	return cmd
}

func newRegistryInitCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Make a directory a registry and register it",
		Long: "Writes a manifest (" + registry.ManifestFile + ") into the directory and adds it to your list of " +
			"registered registries. Agents and tasks already in the directory are adopted unchanged. " +
			"Defaults to the current directory.",
		Args: cobra.MaximumNArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}

			m, err := registry.InitAndRegister(dir, name, time.Now())
			if err != nil {
				return err
			}

			abs, err := filepath.Abs(dir)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "initialised registry %q (id %s) at %s\n", m.Name, m.ID, abs)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "human-readable registry name (defaults to the directory name)")

	return cmd
}

func newRegistryAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Register an existing registry",
		Long: "Adds a registry that already has a manifest (for example, one checked out from version control) " +
			"to your list of registered registries.",
		Args: cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := registry.Register(args[0])
			if errors.Is(err, registry.ErrAlreadyRegistered) {
				fmt.Fprintf(cmd.OutOrStdout(), "already registered: %s\n", args[0])

				return nil
			}
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "registered %q (id %s) at %s\n", m.Name, m.ID, args[0])

			return nil
		},
	}
}

func newRegistryListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered registries",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			paths, err := registry.ListRegistered()
			if err != nil {
				return err
			}

			if len(paths) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no registries registered: run `pi-agent-lab registry init` in a directory, or `pi-agent-lab registry add <path>`")

				return nil
			}

			tw := format.NewTabWriter(cmd.OutOrStdout())
			fmt.Fprintln(tw, "NAME\tPATH\tSTATUS")
			for _, p := range paths {
				name, status := "-", "ok"
				if m, err := registry.ReadManifest(p); err != nil {
					status = "missing or invalid manifest"
				} else {
					name = m.Name
				}

				fmt.Fprintf(tw, "%s\t%s\t%s\n", name, p, status)
			}

			return tw.Flush()
		},
	}
}
