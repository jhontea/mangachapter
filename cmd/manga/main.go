package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	a := &app{}

	root := &cobra.Command{
		Use:   "manga",
		Short: "Monitor manga chapters and send notifications",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return a.init()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			a.close()
		},
	}

	root.PersistentFlags().StringVar(&a.configPath, "config", "", "path to config file (default: config.yaml or MANGA_CONFIG_PATH)")
	root.PersistentFlags().BoolVar(&a.debug, "debug", false, "enable debug logging")

	root.AddCommand(newAddCmd(a))
	root.AddCommand(newListCmd(a))
	root.AddCommand(newRemoveCmd(a))
	root.AddCommand(newSearchCmd(a))
	root.AddCommand(newCheckCmd(a))
	root.AddCommand(newRunCmd(a))

	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}