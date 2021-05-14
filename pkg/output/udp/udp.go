// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package tcp

import (
	"context"
	"net"

	"golang.org/x/time/rate"

	"github.com/elastic/stream/pkg/output"
)

const burst = 1024 * 1024

func init() {
	output.Register("udp", New)
}

type Output struct {
	opts  *output.Options
	conn  *net.UDPConn
	ctx   context.Context
	limit *rate.Limiter
}

func New(opts *output.Options) (output.Output, error) {
	return &Output{
		opts:  opts,
		limit: rate.NewLimiter(rate.Limit(opts.RateLimit), burst),
	}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	udpAddr, err := net.ResolveUDPAddr("udp", o.opts.Addr)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}

	o.conn = conn
	o.ctx = ctx
	return nil
}

func (o *Output) Close() error {
	return o.conn.Close()
}

func (o *Output) Write(b []byte) (int, error) {
	if err := o.limit.WaitN(o.ctx, len(b)); err != nil {
		return 0, err
	}
	return o.conn.Write(b)
}
