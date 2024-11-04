// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

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

type Output struct {
	opts *output.Options
	conn *net.TCPConn
}

func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	d := net.Dialer{Timeout: time.Second}

	conn, err := d.DialContext(ctx, "tcp", o.opts.Addr)
	if err != nil {
		return err
	}

	o.conn = conn.(*net.TCPConn)
	return nil
}

func (o *Output) Conn() net.Conn {
	return o.conn
}

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

func (o *Output) Write(b []byte) (int, error) {
	return o.conn.Write(append(b, '\n'))
}
