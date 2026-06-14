package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Daftar manga yang dilacak",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := a.repo.ListManga(a.context())
			if err != nil {
				return err
			}

			if len(items) == 0 {
				fmt.Println("Belum ada manga yang dilacak.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tSUMBER\tJUDUL\tCHAPTER TERAKHIR\tURL")
			for _, m := range items {
				last := m.LastChapter
				if last == "" {
					last = "-"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", m.ID, m.Source, m.Title, last, m.URL)
			}
			return w.Flush()
		},
	}
}