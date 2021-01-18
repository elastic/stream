package tcp

import (
	"context"
	"net"
	"time"

	"github.com/andrewkroh/stream/pkg/output"
)

func init() {
	output.Register("tcp", New)
}

type Output struct {
	opts *output.Options
	conn net.Conn
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

	o.conn = conn
	return nil
}

func (o *Output) Conn() net.Conn {
	return o.conn
}

func (o *Output) Close() error {
	if o.conn == nil {
		return nil
	}
	return o.conn.Close()
}

func (o *Output) Write(b []byte) (int, error) {
	return o.conn.Write(b)
}
