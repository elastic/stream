// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/elastic/stream/pkg/httpserver"
)

type httpServerRunner struct {
	logger *zap.SugaredLogger
	cmd    *cobra.Command
	opts   *httpserver.Options
}

func newHTTPServerRunner(options *httpserver.Options, logger *zap.Logger) *cobra.Command {
	r := &httpServerRunner{
		opts: options,
		cmd: &cobra.Command{
			Use:   "http-server [options]",
			Short: "Set up a mock http server",
		},
	}

	r.cmd.RunE = func(_ *cobra.Command, args []string) error {
		r.logger = logger.Sugar().With("address", options.Addr)
		return r.Run()
	}

	return r.cmd
}

func (r *httpServerRunner) Run() error {
	r.logger.Debug("mock server running...")
	server, err := httpserver.New(r.opts, r.logger)
	if err != nil {
		return err
	}

	if err := server.Start(r.cmd.Context()); err != nil {
		return err
	}

	<-r.cmd.Context().Done()

	return server.Close()
}
