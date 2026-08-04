// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package command

import (
	"bytes"
	"compress/gzip"
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// helpers

// udpPacket builds an Ethernet/IPv4/UDP frame carrying payload.
func udpPacket(t *testing.T, payload string) []byte {
	t.Helper()

	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
		DstMAC:       net.HardwareAddr{0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		SrcIP:    net.IPv4(10, 0, 0, 1),
		DstIP:    net.IPv4(10, 0, 0, 2),
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{SrcPort: 12345, DstPort: 2055}
	require.NoError(t, udp.SetNetworkLayerForChecksum(ip))

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	require.NoError(t, gopacket.SerializeLayers(buf, opts, eth, ip, udp, gopacket.Payload(payload)))
	return buf.Bytes()
}

// writePCAP writes packets to a classic libpcap capture file and returns its path.
func writePCAP(t *testing.T, packets ...[]byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.pcap")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	w := pcapgo.NewWriter(f)
	require.NoError(t, w.WriteFileHeader(65536, layers.LinkTypeEthernet))
	for _, p := range packets {
		require.NoError(t, w.WritePacket(captureInfo(p), p))
	}
	return path
}

// writePCAPNg writes packets to a pcapng capture file and returns its path.
func writePCAPNg(t *testing.T, packets ...[]byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.pcapng")
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()

	w, err := pcapgo.NewNgWriter(f, layers.LinkTypeEthernet)
	require.NoError(t, err)
	for _, p := range packets {
		require.NoError(t, w.WritePacket(captureInfo(p), p))
	}
	require.NoError(t, w.Flush())
	return path
}

func captureInfo(p []byte) gopacket.CaptureInfo {
	return gopacket.CaptureInfo{
		Timestamp:     time.Unix(1700000000, 0),
		CaptureLength: len(p),
		Length:        len(p),
	}
}

// gzipFile compresses the file at path and returns the path of the new file.
func gzipFile(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	gzPath := path + ".gz"
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write(raw)
	require.NoError(t, err)
	require.NoError(t, gw.Close())
	require.NoError(t, os.WriteFile(gzPath, buf.Bytes(), 0o600))
	return gzPath
}

// memoryOutput is an output.Output that records everything written to it.
type memoryOutput struct {
	writes [][]byte
}

func (*memoryOutput) DialContext(context.Context) error { return nil }
func (*memoryOutput) Close() error                      { return nil }

func (m *memoryOutput) Write(b []byte) (int, error) {
	m.writes = append(m.writes, bytes.Clone(b))
	return len(b), nil
}

// payloads returns the recorded writes as strings.
func (m *memoryOutput) payloads() []string {
	out := make([]string, 0, len(m.writes))
	for _, w := range m.writes {
		out = append(out, string(w))
	}
	return out
}

func newTestRunner(t *testing.T) *pcapRunner {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return &pcapRunner{
		logger: zap.NewNop().Sugar(),
		cmd:    cmd,
	}
}

// tests

func TestNewPacketReader(t *testing.T) {
	packet := udpPacket(t, "hello")

	testCases := []struct {
		name string
		path func(t *testing.T) string
	}{
		{
			name: "pcap",
			path: func(t *testing.T) string { return writePCAP(t, packet) },
		},
		{
			name: "pcapng",
			path: func(t *testing.T) string { return writePCAPNg(t, packet) },
		},
		{
			name: "gzipped pcap",
			path: func(t *testing.T) string { return gzipFile(t, writePCAP(t, packet)) },
		},
		{
			name: "gzipped pcapng",
			path: func(t *testing.T) string { return gzipFile(t, writePCAPNg(t, packet)) },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.path(t))
			require.NoError(t, err)
			defer f.Close()

			source, linkType, err := newPacketReader(f)
			require.NoError(t, err)
			assert.Equal(t, layers.LinkTypeEthernet, linkType)

			data, _, err := source.ReadPacketData()
			require.NoError(t, err)
			assert.Equal(t, packet, data)
		})
	}
}

func TestNewPacketReaderInvalid(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
	}{
		{name: "empty", content: nil},
		{name: "garbage", content: []byte("this is not a capture file")},
		{name: "bad magic", content: []byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x00, 0x00, 0x00}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invalid")
			require.NoError(t, os.WriteFile(path, tc.content, 0o600))

			f, err := os.Open(path)
			require.NoError(t, err)
			defer f.Close()

			_, _, err = newPacketReader(f)
			require.Error(t, err)
		})
	}
}

func TestSendPCAP(t *testing.T) {
	testCases := []struct {
		name string
		path func(t *testing.T) string
	}{
		{
			name: "pcap",
			path: func(t *testing.T) string {
				return writePCAP(t, udpPacket(t, "one"), udpPacket(t, "two"))
			},
		},
		{
			name: "pcapng",
			path: func(t *testing.T) string {
				return writePCAPNg(t, udpPacket(t, "one"), udpPacket(t, "two"))
			},
		},
		{
			name: "gzipped pcap",
			path: func(t *testing.T) string {
				return gzipFile(t, writePCAP(t, udpPacket(t, "one"), udpPacket(t, "two")))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out := &memoryOutput{}
			require.NoError(t, newTestRunner(t).sendPCAP(tc.path(t), out))
			assert.Equal(t, []string{"one", "two"}, out.payloads())
		})
	}
}

// TestSendPCAPSkipsPacketsWithoutTransportLayer verifies that packets carrying
// no transport layer are skipped rather than failing the whole capture.
func TestSendPCAPSkipsPacketsWithoutTransportLayer(t *testing.T) {
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
		DstMAC:       net.HardwareAddr{0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b},
		EthernetType: layers.EthernetTypeARP,
	}
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         1,
		SourceHwAddress:   []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05},
		SourceProtAddress: []byte{10, 0, 0, 1},
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte{10, 0, 0, 2},
	}
	buf := gopacket.NewSerializeBuffer()
	require.NoError(t, gopacket.SerializeLayers(buf, gopacket.SerializeOptions{FixLengths: true}, eth, arp))

	path := writePCAP(t, buf.Bytes(), udpPacket(t, "payload"))

	out := &memoryOutput{}
	require.NoError(t, newTestRunner(t).sendPCAP(path, out))
	assert.Equal(t, []string{"payload"}, out.payloads())
}

// TestSendPCAPTruncated verifies that a capture cut short mid-record streams the
// packets that were readable instead of returning an error.
func TestSendPCAPTruncated(t *testing.T) {
	path := writePCAP(t, udpPacket(t, "first"), udpPacket(t, "second"))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	// Drop the last few bytes so the final record is incomplete.
	truncated := filepath.Join(t.TempDir(), "truncated.pcap")
	require.NoError(t, os.WriteFile(truncated, raw[:len(raw)-8], 0o600))

	out := &memoryOutput{}
	require.NoError(t, newTestRunner(t).sendPCAP(truncated, out))
	assert.Equal(t, []string{"first"}, out.payloads())
}

// TestSendPCAPCorruptRecord verifies that an unparseable record returns an error
// rather than looping forever, which is what gopacket's PacketSource does.
func TestSendPCAPCorruptRecord(t *testing.T) {
	path := writePCAP(t, udpPacket(t, "first"))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	// The second record claims a capture length far beyond the snaplen.
	corrupt := append(bytes.Clone(raw),
		0x00, 0x00, 0x00, 0x00, // timestamp seconds
		0x00, 0x00, 0x00, 0x00, // timestamp microseconds
		0xff, 0xff, 0xff, 0xff, // capture length
		0xff, 0xff, 0xff, 0xff, // original length
	)
	corruptPath := filepath.Join(t.TempDir(), "corrupt.pcap")
	require.NoError(t, os.WriteFile(corruptPath, corrupt, 0o600))

	out := &memoryOutput{}
	err = newTestRunner(t).sendPCAP(corruptPath, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read packet 2")
}

// TestSendPCAPRespectsContextCancellation verifies that a cancelled context
// stops the capture from being streamed.
func TestSendPCAPRespectsContextCancellation(t *testing.T) {
	path := writePCAP(t, udpPacket(t, "one"), udpPacket(t, "two"))

	r := newTestRunner(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r.cmd.SetContext(ctx)

	out := &memoryOutput{}
	require.NoError(t, r.sendPCAP(path, out))
	assert.Empty(t, out.payloads())
}
