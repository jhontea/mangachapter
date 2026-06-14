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
		Short: "Jalankan daemon scheduler (periksa berkala)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := a.init(); err != nil {
				return err
			}
			defer a.close()

			interval := a.cfg.SchedulerInterval()

			// Buat fungsi checker yang mengabaikan hasil
			checkFn := scheduler.CheckAllFunc(func(ctx context.Context) error {
				_, err := a.checker.CheckAll(ctx)
				return err
			})

			s := scheduler.New(checkFn, interval)

			// Tangani SIGINT/SIGTERM
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Jalankan scheduler di goroutine
			go s.Run(ctx)

			// Tunggu sinyal
			<-ctx.Done()
			s.Stop()

			return nil
		},
	}
}