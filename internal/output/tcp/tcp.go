// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package tcp provides an output that writes data to a TCP connection.
package tcp

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/elastic/stream/internal/output"
)

func init() {
	output.Register("tcp", New)
}

// Output is an output that writes to a TCP connection.
type Output struct {
	opts *output.Options
	conn *net.TCPConn
}

// New creates a new TCP output.
func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
}

// DialContext dials the TCP connection.
func (o *Output) DialContext(ctx context.Context) error {
	d := net.Dialer{Timeout: time.Second}

	conn, err := d.DialContext(ctx, "tcp", o.opts.Addr)
	if err != nil {
		return err
	}

	o.conn = conn.(*net.TCPConn)
	return nil
}

// Conn returns the underlying net.Conn.
func (o *Output) Conn() net.Conn {
	return o.conn
}

// Close closes the TCP connection.
func (o *Output) Close() error {
	if o.conn != nil {
		if err := o.conn.CloseWrite(); err != nil {
			return err
		}

		// drain to facilitate graceful close on the other side
		deadline := time.Now().Add(5 * time.Second)
		if err := o.conn.SetReadDeadline(deadline); err != nil {
			return err
		}
		buffer := make([]byte, 1024)
		for {
			_, err := o.conn.Read(buffer)
			if errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				return err
			}
		}

		return o.conn.Close()
	}
	return nil
}

// Write writes data to the TCP connection.
func (o *Output) Write(b []byte) (int, error) {
	if o.conn == nil {
		return 0, errors.New("not connected")
	}

	// Add a newline for framing.
	buf := make([]byte, len(b)+1)
	copy(buf, b)
	buf[len(b)] = '\n'
	return o.conn.Write(buf)
}
