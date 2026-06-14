package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"project/mangachapter/internal/checker"

	"github.com/spf13/cobra"
)

func newCheckCmd(a *app) *cobra.Command {
	var id int64

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check for new chapters",
		Long:  `Check all tracked manga (or a single one with --id) for new chapters.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("id") {
				r, err := a.checker.CheckOne(a.context(), id)
				if err != nil {
					return err
				}
				printResults([]checker.Result{*r})
				return nil
			}

			results, err := a.checker.CheckAll(a.context())
			if err != nil {
				return err
			}
			printResults(results)
			return nil
		},
	}

	cmd.Flags().Int64Var(&id, "id", 0, "check a single manga by ID")
	return cmd
}

func printResults(results []checker.Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSOURCE\tSTATUS\tCHAPTER")

	for _, r := range results {
		status := "OK"
		chapter := "-"
		if r.Error != nil {
			status = "ERROR"
			chapter = r.Error.Error()
		} else if r.NewChapter != "" {
			status = "NEW"
			chapter = r.NewChapter
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", r.MangaID, r.Title, r.Source, status, chapter)
	}
	_ = w.Flush()
}
