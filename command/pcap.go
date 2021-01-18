package command

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/andrewkroh/stream/pkg/output"
)

type pcapRunner struct {
	logger   *zap.SugaredLogger
	cmd      *cobra.Command
	out      *output.Options
	pcapFile string
}

func newPCAPRunner(options *output.Options, logger *zap.Logger) *cobra.Command {
	r := &pcapRunner{
		out: options,
		cmd: &cobra.Command{
			Use:   "pcap [pcap data to stream]",
			Short: "Stream PCAP payload data",
			Args:  cobra.ExactArgs(1),
		},
	}

	r.cmd.RunE = func(_ *cobra.Command, args []string) error {
		r.logger = logger.Sugar().With("address", options.Addr)
		return r.Run(args)
	}

	return r.cmd
}

func (r *pcapRunner) Run(pcapFiles []string) error {
	f, err := pcap.OpenOffline(pcapFiles[0])
	if err != nil {
		return err
	}
	defer f.Close()

	o, err := output.Initialize(r.out, r.logger, r.cmd.Context())
	if err != nil {
		return err
	}
	defer o.Close()

	// Process packets in PCAP and get flow records.
	var totalBytes, totalPackets int
	packetSource := gopacket.NewPacketSource(f, f.LinkType())
	for packet := range packetSource.Packets() {
		if r.cmd.Context().Err() != nil {
			break
		}

		payloadData := packet.TransportLayer().LayerPayload()

		// TODO: Rate-limit for UDP.
		r.logger.Debug("Writing packet")
		n, err := o.Write(payloadData)
		if err != nil {
			return err
		}
		totalBytes += n
		totalPackets++
	}

	r.logger.Infow("Sent data", "sent_bytes", totalBytes, "sent_packets", totalPackets)
	return nil
}
