// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("webhook", New)
}

type Output struct {
	opts   *output.Options
	client *http.Client
}

func New(opts *output.Options) (output.Output, error) {
	if _, err := url.Parse(opts.Addr); err != nil {
		return nil, fmt.Errorf("address must be a valid URL for webhook output: %w", err)
	}

	client := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.InsecureTLS,
			},
		},
	}

	return &Output{opts: opts, client: client}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	// Use a HEAD request to check if the service is ready.
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, o.opts.Addr, nil)
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

func (o *Output) Close() error {
	o.client.CloseIdleConnections()
	return nil
}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("http post to webhook failed with http status %v %v", resp.StatusCode, resp.Status)
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
