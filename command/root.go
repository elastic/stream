package command

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/andrewkroh/stream/pkg/output"

	// Register outputs.
	_ "github.com/andrewkroh/stream/pkg/output/tcp"
	_ "github.com/andrewkroh/stream/pkg/output/tls"
	_ "github.com/andrewkroh/stream/pkg/output/udp"
)

func Execute() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		cancel()
	}()

	return ExecuteContext(ctx)
}

func ExecuteContext(ctx context.Context) error {
	logger, err := logger()
	if err != nil {
		return nil
	}

	rootCmd := &cobra.Command{Use: "stream", SilenceUsage: true}

	// Global flags.
	var outOpts output.Options
	rootCmd.PersistentFlags().StringVar(&outOpts.Addr, "addr", "", "destination address")
	rootCmd.PersistentFlags().DurationVar(&outOpts.Delay, "delay", 0, "delay streaming")
	rootCmd.PersistentFlags().StringVarP(&outOpts.Protocol, "protocol", "p", "tcp", "protocol (tcp/udp/tls)")
	rootCmd.PersistentFlags().IntVar(&outOpts.Retries, "retry", 10, "connection retry attempts")

	// Sub-commands.
	rootCmd.AddCommand(newLogRunner(&outOpts, logger))
	rootCmd.AddCommand(newPCAPRunner(&outOpts, logger))
	rootCmd.AddCommand(versionCmd)
	return rootCmd.ExecuteContext(ctx)
}

func logger() (*zap.Logger, error) {
	conf := zap.NewProductionConfig()
	conf.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	conf.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	log, err := conf.Build()
	if err != nil {
		return nil, err
	}
	return log, nil
}
