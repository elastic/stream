package command

import (
	"bufio"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/andrewkroh/stream/pkg/output"
)

type logRunner struct {
	logger   *zap.SugaredLogger
	cmd      *cobra.Command
	out      *output.Options
	pcapFile string
}

func newLogRunner(options *output.Options, logger *zap.Logger) *cobra.Command {
	r := &logRunner{
		out: options,
		cmd: &cobra.Command{
			Use:   "log [log file to stream]",
			Short: "Stream log file lines",
			Args:  cobra.ExactArgs(1),
		},
	}

	r.cmd.RunE = func(_ *cobra.Command, args []string) error {
		r.logger = logger.Sugar().With("address", options.Addr)
		return r.Run(args)
	}

	return r.cmd
}

func (r *logRunner) Run(files []string) error {
	f, err := os.Open(files[0])
	if err != nil {
		return err
	}
	defer f.Close()

	o, err := output.Initialize(r.out, r.logger, r.cmd.Context())
	if err != nil {
		return err
	}
	defer o.Close()

	var totalBytes, totalLines int
	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		if r.cmd.Context().Err() != nil {
			break
		}

		r.logger.Debug("Writing packet")
		n, err := o.Write(s.Bytes())
		if err != nil {
			return err
		}
		totalBytes += n
		totalLines++
	}
	if s.Err() != nil {
		return s.Err()
	}

	r.logger.Infow("Log data sent", "sent_bytes", totalBytes, "sent_lines", totalLines)
	return nil
}
