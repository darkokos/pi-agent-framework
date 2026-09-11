package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	registry "github.com/darkokos/pi-agent-framework/pi-agent-registry"
)

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and change pi-agent-lab configuration",
	}

	cmd.AddCommand(newConfigSetRegistryDirCommand())
	cmd.AddCommand(newConfigShowCommand())

	return cmd
}

func newConfigSetRegistryDirCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set-registry-dir <path>",
		Short: "Set where the agent/task registry lives",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			abs, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}

			if err := registry.SetRegistryDir(abs); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "registry directory set to %s\n", abs)

			return nil
		},
	}
}

func newConfigShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the resolved registry directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := registry.ResolveRoot()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), root)

			return nil
		},
	}
}
