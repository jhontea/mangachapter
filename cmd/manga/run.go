package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"project/mangachapter/internal/scheduler"
)

func newRunCmd(a *app) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run the scheduler daemon (check periodically)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.init(); err != nil {
				return err
			}
			defer a.close()

			interval := a.cfg.SchedulerInterval()

			// Create a checker function that ignores results
			checkFn := scheduler.CheckAllFunc(func(ctx context.Context) error {
				_, err := a.checker.CheckAll(ctx)
				return err
			})

			s := scheduler.New(checkFn, interval)

			// Handle SIGINT/SIGTERM
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Run scheduler in a goroutine
			go s.Run(ctx)

			// Wait for signal
			<-ctx.Done()
			s.Stop()

			return nil
		},
	}
}