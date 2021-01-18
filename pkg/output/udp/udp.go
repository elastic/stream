package tcp

import (
	"context"
	"net"

	"github.com/andrewkroh/stream/pkg/output"
)

func init() {
	output.Register("udp", New)
}

type Output struct {
	opts *output.Options
	conn *net.UDPConn
}

func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
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
	return nil
}

func (o *Output) Close() error {
	return o.conn.Close()
}

func (o *Output) Write(b []byte) (int, error) {
	return o.conn.Write(b)
}
