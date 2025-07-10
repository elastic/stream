// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package tls provides an output that writes data to a TLS+TCP connection.
package tls

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"

	"github.com/elastic/stream/internal/output"
)

func init() {
	output.Register("tls", New)
}

// Output is an output that writes to a TLS connection.
type Output struct {
	opts *output.Options
	conn *tls.Conn
}

// New creates a new TLS output.
func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
}

// DialContext dials the TLS connection.
func (o *Output) DialContext(ctx context.Context) error {
	d := tls.Dialer{
		Config: &tls.Config{
			InsecureSkipVerify: o.opts.InsecureTLS,
		},
		NetDialer: &net.Dialer{Timeout: time.Second},
	}

	conn, err := d.DialContext(ctx, "tcp", o.opts.Addr)
	if err != nil {
		return err
	}

	o.conn = conn.(*tls.Conn)
	return nil
}

// Close closes the TLS connection.
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

// Write writes data to the TLS connection.
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
