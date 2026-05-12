// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package unix provides an output for writing to unix sockets
package unix

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/elastic/stream/internal/output"
)

func init() {
	output.Register("unix", New)
}

// Output holds options and connection
type Output struct {
	opts *output.Options
	conn *net.UnixConn
}

// New creates a new unix output
func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
}

// DialContext connects to the address in the Output struct using the supplied context
func (o *Output) DialContext(ctx context.Context) error {
	d := net.Dialer{Timeout: time.Second}

	conn, err := d.DialContext(ctx, "unix", o.opts.Addr)
	if err != nil {
		return err
	}

	o.conn = conn.(*net.UnixConn)
	return nil
}

// Conn returns the connection
func (o *Output) Conn() net.Conn {
	return o.conn
}

// Close gracefully closes the connection
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

// Write the supplied bytes to the connection and appends a newline
// character.  The adding of the newline character is to behave the
// same as the tcp output.
func (o *Output) Write(b []byte) (int, error) {
	return o.conn.Write(append(b, '\n')) //nolint:staticcheck  // convention established in tcp output
}
