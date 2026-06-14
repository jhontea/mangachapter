package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRemoveCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a manga from the watchlist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented: remove %s", args[0])
		},
	}
}
