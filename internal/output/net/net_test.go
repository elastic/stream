// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package netout

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/stream/internal/output"
)

// helpers

func newTCPListener(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	return l
}

func newUnixListener(t *testing.T) net.Listener {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.sock")
	l, err := net.Listen("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	return l
}

// acceptAndDrain accepts one connection, drains it, and closes it.
func acceptAndDrain(l net.Listener) {
	conn, err := l.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	buf := make([]byte, 4096)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			break
		}
	}
}

// acceptAndCollect accepts one connection, reads one chunk, and sends it on ch.
func acceptAndCollect(l net.Listener, ch chan<- []byte) {
	conn, err := l.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	buf := make([]byte, 4096)
	n, _ := conn.Read(buf)
	ch <- buf[:n]
}

// TCP tests

func TestTCPDial(t *testing.T) {
	l := newTCPListener(t)
	go acceptAndDrain(l)

	out, err := New(&output.Options{Protocol: "tcp", Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))
	require.NoError(t, out.Close())
}

func TestTCPDialInvalid(t *testing.T) {
	out, err := New(&output.Options{Protocol: "tcp", Addr: "127.0.0.1:1"})
	require.NoError(t, err)
	require.Error(t, out.DialContext(context.Background()))
}

func TestTCPWrite(t *testing.T) {
	l := newTCPListener(t)
	ch := make(chan []byte, 1)
	go acceptAndCollect(l, ch)

	out, err := New(&output.Options{Protocol: "tcp", Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))

	msg := []byte("hello tcp")
	_, err = out.Write(msg)
	require.NoError(t, err)
	assert.Equal(t, append(msg, '\n'), <-ch)
}

func TestTCPCloseWithoutDial(t *testing.T) {
	out, err := New(&output.Options{Protocol: "tcp", Addr: "127.0.0.1:1"})
	require.NoError(t, err)
	require.NoError(t, out.Close())
}

// Unix socket tests

func TestUnixDial(t *testing.T) {
	l := newUnixListener(t)
	go acceptAndDrain(l)

	out, err := New(&output.Options{Protocol: "unix", Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))
	require.NoError(t, out.Close())
}

func TestUnixDialInvalidPath(t *testing.T) {
	out, err := New(&output.Options{Protocol: "unix", Addr: "/nonexistent/path/test.sock"})
	require.NoError(t, err)
	require.Error(t, out.DialContext(context.Background()))
}

func TestUnixWrite(t *testing.T) {
	l := newUnixListener(t)
	ch := make(chan []byte, 1)
	go acceptAndCollect(l, ch)

	out, err := New(&output.Options{Protocol: "unix", Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))

	msg := []byte("hello unix")
	_, err = out.Write(msg)
	require.NoError(t, err)
	assert.Equal(t, append(msg, '\n'), <-ch)
}

func TestUnixWriteAppendsNewline(t *testing.T) {
	l := newUnixListener(t)
	ch := make(chan []byte, 1)
	go acceptAndCollect(l, ch)

	out, err := New(&output.Options{Protocol: "unix", Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))

	_, err = out.Write([]byte("no newline"))
	require.NoError(t, err)
	got := <-ch
	assert.Equal(t, byte('\n'), got[len(got)-1])
}

func TestUnixCloseWithoutDial(t *testing.T) {
	out, err := New(&output.Options{Protocol: "unix", Addr: "/tmp/ignored.sock"})
	require.NoError(t, err)
	require.NoError(t, out.Close())
}

// UDP tests

func TestUDPDial(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer pc.Close()

	out, err := New(&output.Options{Protocol: "udp", Addr: pc.LocalAddr().String(), RateLimit: 1024 * 1024})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))
	require.NoError(t, out.Close())
}

func TestUDPWrite(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer pc.Close()

	out, err := New(&output.Options{Protocol: "udp", Addr: pc.LocalAddr().String(), RateLimit: 1024 * 1024})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))

	msg := []byte("hello udp")
	_, err = out.Write(msg)
	require.NoError(t, err)

	buf := make([]byte, 4096)
	n, _, err := pc.ReadFrom(buf)
	require.NoError(t, err)
	// UDP does not append a newline.
	assert.Equal(t, msg, buf[:n])
}

// Unknown protocol test

func TestDialUnknownProtocol(t *testing.T) {
	out, err := New(&output.Options{Protocol: "ftp", Addr: "127.0.0.1:21"})
	require.NoError(t, err)
	err = out.DialContext(context.Background())
	require.ErrorContains(t, err, "unknown protocol")
}

// Registration tests

func TestRegistered(t *testing.T) {
	available := output.Available()
	for _, proto := range []string{"tcp", "tls", "udp", "unix"} {
		assert.Contains(t, available, proto, "protocol %q should be registered", proto)
	}
}
