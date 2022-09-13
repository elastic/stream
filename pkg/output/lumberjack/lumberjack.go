// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package tcp

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	v2 "github.com/elastic/go-lumber/client/v2"

	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("lumberjack", New)
}

type Output struct {
	opts    *output.Options
	scheme  string
	address string
	client  *v2.SyncClient
}

func New(opts *output.Options) (output.Output, error) {
	scheme, host, port, err := splitAddress(opts.Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addr for lumberjack: %w", err)
	}

	return &Output{
		opts:    opts,
		scheme:  scheme,
		address: net.JoinHostPort(host, port),
	}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	var dialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)
	switch o.scheme {
	case "tcp":
		dialer := &net.Dialer{Timeout: time.Second}
		dialContextFunc = dialer.DialContext
	case "tls":
		dialer := &tls.Dialer{
			Config: &tls.Config{
				InsecureSkipVerify: o.opts.InsecureTLS,
			},
			NetDialer: &net.Dialer{Timeout: time.Second},
		}
		dialContextFunc = dialer.DialContext
	default:
		panic("unhandled scheme " + o.scheme)
	}

	dial := func(network, address string) (net.Conn, error) {
		return dialContextFunc(ctx, network, address)
	}

	client, err := v2.SyncDialWith(dial, o.address)
	if err != nil {
		return err
	}

	o.client = client
	return nil
}

func (o *Output) Close() error {
	if o.client != nil {
		return o.client.Close()
	}
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	_, err := o.client.Send(makeBatch(b, o.opts.LumberjackOptions.ParseJSON))
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func splitAddress(addr string) (scheme, host, port string, err error) {
	// Use tcp:// scheme by default if not specified.
	if !strings.Contains(addr, "://") {
		addr = "tcp://" + addr
	}

	url, err := url.Parse(addr)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid address: %w", err)
	}

	// Require an explicit port in addresses.
	if url.Port() == "" {
		return "", "", "", errors.New("port number is required")
	}

	switch url.Scheme {
	case "tcp", "tls":
	default:
		return "", "", "", fmt.Errorf("invalid scheme %q (use tcp or tls)", url.Scheme)
	}

	return url.Scheme, url.Hostname(), url.Port(), nil
}

func makeBatch(b []byte, parseJSON bool) []interface{} {
	if parseJSON {
		return makeBatchFromJSON(b)
	}

	return []interface{}{
		map[string]interface{}{
			"message": string(b),
		},
	}
}

func makeBatchFromJSON(b []byte) []interface{} {
	enc := json.NewDecoder(bytes.NewReader(b))
	enc.UseNumber()

	var data interface{}
	if err := enc.Decode(&data); err != nil {
		return []interface{}{
			map[string]interface{}{
				"message": string(b),
				"tags": []string{
					"invalid-json",
				},
			},
		}
	}

	if slice, ok := data.([]interface{}); ok {
		return slice
	}

	return []interface{}{
		data,
	}
}
