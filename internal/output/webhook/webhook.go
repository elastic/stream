// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package webhook provides an output that sends data to an HTTP or HTTPS
// endpoint via configurable webhooks. It supports customizable HTTP headers,
// basic authentication, custom content types, and configurable TLS settings for
// secure communication. This package is intended to enable sending events or log
// lines to web services that accept data over HTTP, often used for integrations
// or alerting.
package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/stream/internal/output"
)

func init() {
	output.Register("webhook", New)
}

// Output is a webhook output.
type Output struct {
	opts   *output.Options
	client *http.Client
}

// New returns a new webhook output.
func New(opts *output.Options) (output.Output, error) {
	if _, err := url.Parse(opts.Addr); err != nil {
		return nil, fmt.Errorf("address must be a valid URL for webhook output: %w", err)
	}

	if opts.Timeout < 0 {
		return nil, fmt.Errorf("timeout must not be negative: %v", opts.Timeout)
	}
	client := &http.Client{
		Timeout: opts.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureTLS,
			},
		},
	}

	return &Output{opts: opts, client: client}, nil
}

// DialContext connects to the configured endpoint.
func (o *Output) DialContext(ctx context.Context) error {
	method := o.opts.WebhookOptions.Probe
	switch method {
	case "", "1", "true", http.MethodHead:
		// Default behaviour is to do a HEAD probe.
		method = http.MethodHead
	case "0", "false":
		// Don't probe.
		return nil
	case http.MethodGet, http.MethodConnect, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut:
		// Fall through with the option that the env var specifies.
	default:
		return fmt.Errorf("unknown probe behavior option: %q", method)
	}

	// Use a request to check if the service is ready.
	req, err := http.NewRequestWithContext(ctx, method, o.opts.Addr, nil)
	if err != nil {
		return err
	}

	if o.opts.Username != "" && o.opts.Password != "" {
		req.SetBasicAuth(o.opts.Username, o.opts.Password)
	}
	if err = setHeaders(req, o.opts.Headers); err != nil {
		return err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Don't check the status code in case the endpoint does not support HEAD.
	return nil
}

// Close closes the connection to the configured endpoint.
func (o *Output) Close() error {
	o.client.CloseIdleConnections()
	return nil
}

// Write writes data to the configured endpoint.
func (o *Output) Write(b []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, o.opts.Addr, bytes.NewReader(b))
	if err != nil {
		return 0, err
	}

	if o.opts.ContentType != "" {
		req.Header.Set("Content-Type", o.opts.ContentType)
	}
	if o.opts.Username != "" && o.opts.Password != "" {
		req.SetBasicAuth(o.opts.Username, o.opts.Password)
	}
	if err = setHeaders(req, o.opts.Headers); err != nil {
		return 0, err
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return 0, err
	}
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if buf.Len() == 0 {
			buf.WriteString("no body")
		}
		return 0, fmt.Errorf("http post to webhook failed with http status %v %v: %s", resp.StatusCode, resp.Status, &buf)
	}

	return len(b), nil
}

func setHeaders(req *http.Request, headers []string) error {
	for _, h := range headers {
		parts := strings.SplitN(h, "=", 2)

		switch len(parts) {
		case 2:
			req.Header.Set(parts[0], parts[1])
		default:
			return fmt.Errorf("failed to parse header %q", h)
		}
	}

	return nil
}
