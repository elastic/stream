// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/elastic/go-concert/ctxtool/osctx"
	"github.com/elastic/go-concert/timed"

	"github.com/elastic/stream/pkg/httpserver"
	"github.com/elastic/stream/pkg/log"
	"github.com/elastic/stream/pkg/output"

	// Register outputs.
	_ "github.com/elastic/stream/pkg/output/azureblobstorage"
	_ "github.com/elastic/stream/pkg/output/gcppubsub"
	_ "github.com/elastic/stream/pkg/output/gcs"
	_ "github.com/elastic/stream/pkg/output/kafka"
	_ "github.com/elastic/stream/pkg/output/lumberjack"
	_ "github.com/elastic/stream/pkg/output/tcp"
	_ "github.com/elastic/stream/pkg/output/tls"
	_ "github.com/elastic/stream/pkg/output/udp"
	_ "github.com/elastic/stream/pkg/output/webhook"
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
	logger, err := log.NewLogger()
	if err != nil {
		return nil
	}

	rootCmd := &cobra.Command{Use: "stream", SilenceUsage: true}

	// Global flags.
	var opts output.Options
	rootCmd.PersistentFlags().StringVar(&opts.Addr, "addr", "", "destination address")
	rootCmd.PersistentFlags().DurationVar(&opts.Delay, "delay", 0, "delay start after start-signal")
	rootCmd.PersistentFlags().StringVarP(&opts.Protocol, "protocol", "p", "tcp", "protocol ("+strings.Join(output.Available(), "/")+")")
	rootCmd.PersistentFlags().IntVar(&opts.Retries, "retry", 10, "connection retry attempts for tcp based protocols")
	rootCmd.PersistentFlags().StringVarP(&opts.StartSignal, "start-signal", "s", "", "wait for start signal")
	rootCmd.PersistentFlags().BoolVar(&opts.InsecureTLS, "insecure", false, "disable tls verification")
	rootCmd.PersistentFlags().IntVar(&opts.RateLimit, "rate-limit", 500*1024, "bytes per second rate limit for UDP output")
	rootCmd.PersistentFlags().IntVar(&opts.MaxLogLineSize, "max-log-line-size", 500*1024, "max size of a single log line in bytes")

	// Webhook output flags.
	rootCmd.PersistentFlags().StringVar(&opts.WebhookOptions.ContentType, "webhook-content-type", "application/json", "webhook Content-Type")
	rootCmd.PersistentFlags().StringArrayVar(&opts.WebhookOptions.Headers, "webhook-header", nil, "webhook header to add to request (e.g. Header=Value)")
	rootCmd.PersistentFlags().StringVar(&opts.WebhookOptions.Password, "webhook-password", "", "webhook password for basic authentication")
	rootCmd.PersistentFlags().StringVar(&opts.WebhookOptions.Username, "webhook-username", "", "webhook username for basic authentication")

	// GCP Pubsub output flags.
	rootCmd.PersistentFlags().StringVar(&opts.GCPPubsubOptions.Project, "gcppubsub-project", "test", "GCP Pubsub project name")
	rootCmd.PersistentFlags().StringVar(&opts.GCPPubsubOptions.Topic, "gcppubsub-topic", "topic", "GCP Pubsub topic name")
	rootCmd.PersistentFlags().StringVar(&opts.GCPPubsubOptions.Subscription, "gcppubsub-subscription", "subscription", "GCP Pubsub subscription name")
	rootCmd.PersistentFlags().BoolVar(&opts.GCPPubsubOptions.Clear, "gcppubsub-clear", true, "GCP Pubsub clear flag")

	// GCS output flags.
	rootCmd.PersistentFlags().StringVar(&opts.AzureBlobStorageOptions.Container, "azure-blob-storage-container", "testcontainer", "Azure Blob Storage container name")
	rootCmd.PersistentFlags().StringVar(&opts.AzureBlobStorageOptions.Blob, "azure-blob-storage-blob", "testblob", "Azure Blob Storage blob name")
	rootCmd.PersistentFlags().StringVar(&opts.AzureBlobStorageOptions.Port, "azure-blob-storage-port", "10000", "HTTP port used to connect to the blob storage, used for emulators and CI")

	// Kafka Pubsub output flags.
	rootCmd.PersistentFlags().StringVar(&opts.KafkaOptions.Topic, "kafka-topic", "test", "Kafka topic name")

	// GCS output flags.
	rootCmd.PersistentFlags().StringVar(&opts.GcsOptions.Bucket, "gcs-bucket", "testbucket", "GCS Bucket name")
	rootCmd.PersistentFlags().StringVar(&opts.GcsOptions.Object, "gcs-object", "testobject", "GCS Object name")
	rootCmd.PersistentFlags().StringVar(&opts.GcsOptions.Object, "gcs-content-type", "application/json", "The Content type of the object to be uploaded to GCS.")
	rootCmd.PersistentFlags().StringVar(&opts.GcsOptions.ProjectID, "gcs-projectid", "testproject", "GCS Project name")

	// Lumberjack output flags.
	rootCmd.PersistentFlags().BoolVar(&opts.LumberjackOptions.ParseJSON, "lumberjack-parse-json", false, "Parse the input data as JSON and send the structured data as a Lumberjack batch.")

	// Sub-commands.
	rootCmd.AddCommand(newLogRunner(&opts, logger))
	rootCmd.AddCommand(newPCAPRunner(&opts, logger))

	httpOpts := httpserver.Options{Options: &opts}
	httpCommand := newHTTPServerRunner(&httpOpts, logger)
	httpCommand.PersistentFlags().DurationVar(&httpOpts.ReadTimeout, "read-timeout", 5*time.Second, "HTTP Server read timeout")
	httpCommand.PersistentFlags().DurationVar(&httpOpts.WriteTimeout, "write-timeout", 5*time.Second, "HTTP Server write timeout")
	httpCommand.PersistentFlags().StringVar(&httpOpts.TLSCertificate, "tls-cert", "", "Path to the TLS certificate")
	httpCommand.PersistentFlags().StringVar(&httpOpts.TLSKey, "tls-key", "", "Path to the TLS key file")
	httpCommand.PersistentFlags().StringVar(&httpOpts.ConfigPath, "config", "", "Path to the config file")
	rootCmd.AddCommand(httpCommand)

	rootCmd.AddCommand(versionCmd)

	// Add common start-up delay logic.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return multierr.Combine(
			waitForStartSignal(&opts, cmd.Context(), logger),
			waitForDelay(&opts, cmd.Context(), logger),
		)
	}

	// Automatically set flags based on environment variables.
	rootCmd.PersistentFlags().VisitAll(setFlagFromEnv)

	return rootCmd.ExecuteContext(ctx)
}

func waitForStartSignal(opts *output.Options, parent context.Context, logger *zap.Logger) error {
	if opts.StartSignal == "" {
		return nil
	}

	num := unix.SignalNum(opts.StartSignal)
	if num == 0 {
		return fmt.Errorf("unknown signal %v", opts.StartSignal)
	}

	// Wait for the signal or the command context to be done.
	logger.Sugar().Infow("Waiting for signal.", "start-signal", opts.StartSignal)
	startCtx, _ := osctx.WithSignal(parent, os.Signal(num))
	<-startCtx.Done()
	return nil
}

func waitForDelay(opts *output.Options, parent context.Context, logger *zap.Logger) error {
	if opts.Delay <= 0 {
		return nil
	}

	logger.Sugar().Infow("Delaying connection.", "delay", opts.Delay)
	if err := timed.Wait(parent, opts.Delay); err != nil {
		return fmt.Errorf("delay waiting period was interrupted: %w", err)
	}
	return nil
}

func setFlagFromEnv(flag *pflag.Flag) {
	envVar := strings.ToUpper(flag.Name)
	envVar = strings.ReplaceAll(envVar, "-", "_")
	envVar = "STREAM_" + envVar

	flag.Usage = fmt.Sprintf("%v [env %v]", flag.Usage, envVar)
	if value := os.Getenv(envVar); value != "" {
		flag.Value.Set(value)
	}
}
