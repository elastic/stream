// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/elastic/stream/internal/cmdutil"
	"github.com/elastic/stream/internal/output"
)

type pcapRunner struct {
	logger *zap.SugaredLogger
	cmd    *cobra.Command
	out    *output.Options
}

func newPCAPRunner(options *output.Options, logger *zap.Logger) *cobra.Command {
	r := &pcapRunner{
		out: options,
		cmd: &cobra.Command{
			Use:   "pcap [pcap data to stream]",
			Short: "Stream PCAP payload data",
			Args:  cmdutil.ValidateArgs(cobra.MinimumNArgs(1), cmdutil.RegularFiles),
		},
	}

	r.cmd.RunE = func(_ *cobra.Command, args []string) error {
		r.logger = logger.Sugar().With("address", options.Addr)
		return r.Run(args)
	}

	return r.cmd
}

func (r *pcapRunner) Run(files []string) error {
	out, err := output.Initialize(r.out, r.logger, r.cmd.Context())
	if err != nil {
		return err
	}
	defer out.Close()

	for _, f := range files {
		if err := r.sendPCAP(f, out); err != nil {
			return err
		}
	}

	return nil
}

func (r *pcapRunner) sendPCAP(path string, out output.Output) error {
	logger := r.logger.With("pcap", path)

	f, err := pcap.OpenOffline(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Process packets in PCAP and get flow records.
	var totalBytes, totalPackets int
	packetSource := gopacket.NewPacketSource(f, f.LinkType())
	for packet := range packetSource.Packets() {
		if r.cmd.Context().Err() != nil {
			break
		}

		if packet == nil {
			logger.Warnw("Skipping nil packet")
			continue
		}

		tl := packet.TransportLayer()
		if tl == nil {
			logger.Warnw("Skipping packet with no transport layer")
			continue
		}

		payloadData := tl.LayerPayload()

		// TODO: Rate-limit for UDP.
		n, err := out.Write(payloadData)
		if err != nil {
			return err
		}
		totalBytes += n
		totalPackets++
	}

	logger.Infow("Sent PCAP payload data", "total_bytes", totalBytes, "total_packets", totalPackets)
	return nil
}
