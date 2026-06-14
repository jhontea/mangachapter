package main

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newRemoveCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Hapus manga dari daftar pantau",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("ID tidak valid: %s", args[0])
			}

			if err := a.repo.RemoveManga(a.context(), id); err != nil {
				return fmt.Errorf("hapus manga: %w", err)
			}

			fmt.Printf("Manga dengan ID %d telah dihapus.\n", id)
			return nil
		},
	}
}