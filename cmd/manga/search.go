package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"project/mangachapter/internal/source"

	"github.com/spf13/cobra"
)

func newSearchCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "search <source> <query>",
		Short: "Cari manga di situs sumber",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourceName, query := args[0], args[1]

			src, ok := source.Get(sourceName)
			if !ok {
				return fmt.Errorf("sumber tidak dikenal %q; tersedia: %s", sourceName, strings.Join(source.Available(), ", "))
			}

			fmt.Printf("Mencari %q untuk %q...\n", sourceName, query)
			results, err := src.Search(a.context(), query)
			if err != nil {
				return fmt.Errorf("pencarian: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("Tidak ada hasil ditemukan.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "#\tJUDUL\tURL")
			for i, r := range results {
				fmt.Fprintf(w, "%d\t%s\t%s\n", i+1, r.Title, r.URL)
			}
			return w.Flush()
		},
	}
}