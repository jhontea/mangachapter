package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRunCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the scheduler daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented: scheduler daemon")
		},
	}
}
