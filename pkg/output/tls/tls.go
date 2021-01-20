// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package tcp

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/andrewkroh/stream/pkg/output"
)

func init() {
	output.Register("tls", New)
}

type Output struct {
	opts *output.Options
	conn net.Conn
}

func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	d := tls.Dialer{
		Config: &tls.Config{
			InsecureSkipVerify: true,
		},
		NetDialer: &net.Dialer{Timeout: time.Second},
	}

	conn, err := d.DialContext(ctx, "tcp", o.opts.Addr)
	if err != nil {
		return err
	}

	o.conn = conn
	return nil
}

func (o *Output) Close() error {
	if o.conn != nil {
		return o.conn.Close()
	}
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	return o.conn.Write(b)
}
