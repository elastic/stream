// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
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

// Run executes the pcap command.
func (r *pcapRunner) Run(files []string) error {
	out, err := output.Initialize(r.cmd.Context(), r.out, r.logger)
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

// pcapNgMagic is the block type of the Section Header Block that begins every
// pcapng file. It is palindromic, so it identifies the format regardless of the
// byte order the file was written in.
var pcapNgMagic = []byte{0x0a, 0x0d, 0x0d, 0x0a}

// gzipMagic is the two byte header that begins every gzip stream.
var gzipMagic = []byte{0x1f, 0x8b}

// newPacketReader returns a packet source for the pcap or pcapng data in r,
// along with the link type of its packets. Gzip compressed captures are
// decompressed transparently.
func newPacketReader(r io.Reader) (gopacket.PacketDataSource, layers.LinkType, error) {
	br := bufio.NewReader(r)

	if magic, err := br.Peek(len(gzipMagic)); err == nil && bytes.Equal(magic, gzipMagic) {
		gzipReader, err := gzip.NewReader(br)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to open gzip stream: %w", err)
		}
		br = bufio.NewReader(gzipReader)
	}

	magic, err := br.Peek(len(pcapNgMagic))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read capture file header: %w", err)
	}

	if bytes.Equal(magic, pcapNgMagic) {
		opts := pcapgo.DefaultNgReaderOptions
		// The defaults already mirror libpcap: the link type comes from the
		// first interface and packets from interfaces with a differing link
		// type are ignored. Skipping unknown section versions is recommended by
		// the pcapng spec and matches libpcap, which would otherwise leave a
		// capture containing a newer section unreadable.
		opts.SkipUnknownVersion = true

		reader, err := pcapgo.NewNgReader(br, opts)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid pcapng header: %w", err)
		}
		return reader, reader.LinkType(), nil
	}

	reader, err := pcapgo.NewReader(br)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid pcap header: %w", err)
	}
	return reader, reader.LinkType(), nil
}

func (r *pcapRunner) sendPCAP(path string, out output.Output) error {
	logger := r.logger.With("pcap", path)

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	source, linkType, err := newPacketReader(f)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Process packets in PCAP and get flow records.
	var totalBytes, totalPackets int
readPackets:
	for r.cmd.Context().Err() == nil {
		data, _, err := source.ReadPacketData()
		switch {
		case err == nil:
		case errors.Is(err, io.EOF):
			// End of the capture.
			break readPackets
		case errors.Is(err, io.ErrUnexpectedEOF):
			// Tolerate truncated captures and stream what was readable.
			logger.Warnw("Capture file is truncated, stopping early", "total_packets", totalPackets)
			break readPackets
		default:
			return fmt.Errorf("failed to read packet %d from %s: %w", totalPackets+1, path, err)
		}

		packet := gopacket.NewPacket(data, linkType, gopacket.Default)

		tl := packet.TransportLayer()
		if tl == nil {
			logger.Warnw("Skipping packet with no transport layer")
			continue
		}

		payloadData := tl.LayerPayload()

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
