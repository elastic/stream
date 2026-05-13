// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package netout provides a unified network output supporting tcp, tls, udp, and unix protocols.
package netout

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/stream/internal/output"
)

const burst = 1024 * 1024

func init() {
	output.Register("tcp", New)
	output.Register("tls", New)
	output.Register("udp", New)
	output.Register("unix", New)
}

// Output holds options and the active connection.
type Output struct {
	opts  *output.Options
	conn  net.Conn
	ctx   context.Context
	limit *rate.Limiter
}

// New creates a new network output for the protocol specified in opts.Protocol.
func New(opts *output.Options) (output.Output, error) {
	o := &Output{opts: opts}
	if opts.Protocol == "udp" {
		o.limit = rate.NewLimiter(rate.Limit(opts.RateLimit), burst)
	}
	return o, nil
}

// DialContext connects to the address in opts using the protocol in opts.
func (o *Output) DialContext(ctx context.Context) error {
	var (
		conn net.Conn
		err  error
	)

	switch o.opts.Protocol {
	case "tls":
		d := tls.Dialer{
			Config:    &tls.Config{InsecureSkipVerify: o.opts.InsecureTLS}, //nolint:gosec
			NetDialer: &net.Dialer{Timeout: time.Second},
		}
		conn, err = d.DialContext(ctx, "tcp", o.opts.Addr)
	case "udp":
		conn, err = net.Dial("udp", o.opts.Addr)
		o.ctx = ctx
	case "tcp", "unix":

		d := net.Dialer{Timeout: time.Second}
		conn, err = d.DialContext(ctx, o.opts.Protocol, o.opts.Addr)
	default:
		return fmt.Errorf("unknown protocol: %s", o.opts.Protocol)
	}

	if err != nil {
		return err
	}
	o.conn = conn
	return nil
}

// Close closes the connection. For stream-oriented protocols (tcp, tls, unix) it
// performs a graceful shutdown by signalling EOF and draining remaining data.
func (o *Output) Close() error {
	if o.conn == nil {
		return nil
	}

	if o.opts.Protocol == "udp" {
		return o.conn.Close()
	}

	// Signal EOF to the remote end.
	type closeWriter interface {
		CloseWrite() error
	}
	if cw, ok := o.conn.(closeWriter); ok {
		if err := cw.CloseWrite(); err != nil {
			return err
		}
	}

	// Drain to facilitate graceful close on the other side.
	deadline := time.Now().Add(5 * time.Second)
	if err := o.conn.SetReadDeadline(deadline); err != nil {
		return err
	}
	buf := make([]byte, 1024)
	for {
		_, err := o.conn.Read(buf)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return err
		}
	}

	return o.conn.Close()
}

// Write writes b to the connection. A newline is appended for stream-oriented
// protocols (tcp, tls, unix); UDP datagrams are written as-is.
func (o *Output) Write(b []byte) (int, error) {
	if o.opts.Protocol == "udp" {
		if err := o.limit.WaitN(o.ctx, len(b)); err != nil {
			return 0, err
		}
		return o.conn.Write(b)
	}
	return o.conn.Write(append(b, '\n'))
}
