// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package unix

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/stream/internal/output"
)

func newListener(t *testing.T) *net.UnixListener {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.sock")
	l, err := net.Listen("unix", path)
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })
	return l.(*net.UnixListener)
}

func TestDial(t *testing.T) {
	l := newListener(t)

	// Accept and drain so Close()'s graceful shutdown can complete.
	go func() {
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
	}()

	out, err := New(&output.Options{Addr: l.Addr().String()})
	require.NoError(t, err)

	err = out.DialContext(context.Background())
	require.NoError(t, err)
	require.NoError(t, out.Close())
}

func TestDialInvalidPath(t *testing.T) {
	out, err := New(&output.Options{Addr: "/nonexistent/path/test.sock"})
	require.NoError(t, err)

	err = out.DialContext(context.Background())
	require.Error(t, err)
}

func TestWrite(t *testing.T) {
	l := newListener(t)

	// Accept one connection and read all data from it.
	received := make(chan []byte, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	out, err := New(&output.Options{Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))

	msg := []byte("hello world")
	_, err = out.Write(msg)
	require.NoError(t, err)

	got := <-received
	assert.Equal(t, append(msg, '\n'), got)
}

func TestWriteAppendsNewline(t *testing.T) {
	l := newListener(t)

	received := make(chan []byte, 1)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		received <- buf[:n]
	}()

	out, err := New(&output.Options{Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))

	_, err = out.Write([]byte("no newline"))
	require.NoError(t, err)

	got := <-received
	assert.Equal(t, byte('\n'), got[len(got)-1])
}

func TestClose(t *testing.T) {
	l := newListener(t)

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := l.Accept()
		if err != nil {
			return
		}
		// Drain until EOF so the graceful close can complete.
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				break
			}
		}
		conn.Close()
	}()

	out, err := New(&output.Options{Addr: l.Addr().String()})
	require.NoError(t, err)
	require.NoError(t, out.DialContext(context.Background()))
	require.NoError(t, out.Close())
	<-done
}

func TestCloseWithoutDial(t *testing.T) {
	out, err := New(&output.Options{Addr: "/tmp/ignored.sock"})
	require.NoError(t, err)
	require.NoError(t, out.Close())
}

func TestRegistered(t *testing.T) {
	outputs := output.Available()
	assert.Contains(t, outputs, "unix")
}
