// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/elastic/stream/internal/cmdutil"
	"github.com/elastic/stream/internal/output"
	"github.com/elastic/stream/internal/templates"
)

type logRunner struct {
	logger   *zap.SugaredLogger
	cmd      *cobra.Command
	out      *output.Options
	template bool
}

func newLogRunner(options *output.Options, logger *zap.Logger) *cobra.Command {
	r := &logRunner{
		out: options,
		cmd: &cobra.Command{
			Use:   "log [log file to stream]",
			Short: "Stream log file lines",
			Args:  cmdutil.ValidateArgs(cobra.MinimumNArgs(1), cmdutil.RegularFiles),
		},
	}

	r.cmd.PersistentFlags().BoolVar(&r.template, "template", false,
		"evaluate each log line as a Go text/template before sending, using the same functions as the http-server config (e.g. now)")

	r.cmd.RunE = func(_ *cobra.Command, args []string) error {
		r.logger = logger.Sugar().With("address", options.Addr)
		return r.Run(args)
	}

	return r.cmd
}

// Run executes the log command.
func (r *logRunner) Run(args []string) error {
	out, err := output.Initialize(r.cmd.Context(), r.out, r.logger)
	if err != nil {
		return err
	}
	defer out.Close()

	files, err := cmdutil.ExpandGlobPatternsFromArgs(args)
	if err != nil {
		return err
	}

	for _, f := range files {
		if err := r.sendLog(f, out); err != nil {
			return err
		}
	}

	return nil
}

func (r *logRunner) sendLog(path string, out output.Output) error {
	logger := r.logger.With("log", path)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var totalBytes, totalLines int
	s := bufio.NewScanner(bufio.NewReader(f))
	buf := make([]byte, r.out.MaxLogLineSize)
	s.Buffer(buf, r.out.MaxLogLineSize)
	for s.Scan() {
		if r.cmd.Context().Err() != nil {
			break
		}

		logger.Debugw("Sending log line.", "line_number", totalLines+1)

		line := s.Bytes()
		if r.template {
			rendered, err := templates.Render(s.Text())
			if err != nil {
				return fmt.Errorf("failed to render template on line %d of %s: %w", totalLines+1, path, err)
			}
			line = rendered
		}

		n, err := out.Write(line)
		if err != nil {
			return err
		}

		totalBytes += n
		totalLines++
	}
	if s.Err() != nil {
		return s.Err()
	}

	logger.Infow("Log data sent.", "total_bytes", totalBytes, "total_lines", totalLines)
	return nil
}
