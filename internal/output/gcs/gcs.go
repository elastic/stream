// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package gcs provides an output implementation for streaming data to Google
// Cloud Storage (GCS) buckets. It handles the connection setup, bucket creation
// (if it does not exist), and writing data as objects within the specified
// bucket.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/elastic/stream/internal/output"
)

func init() {
	output.Register("gcs", New)
}

// Output is a GCS output.
type Output struct {
	opts       *output.Options
	client     *storage.Client
	writer     *storage.Writer
	cancelFunc func()
}

// New returns a new GCS output.
func New(opts *output.Options) (output.Output, error) {
	gcsClient, ctx, cancel, err := NewClient(opts.Addr)
	if err != nil {
		return nil, err
	}
	obj := gcsClient.Bucket(opts.GCSOptions.Bucket).Object(opts.GCSOptions.Object)
	writer := obj.NewWriter(ctx)
	// System tests are failing because a default content type is not set automatically, so we set it here instead.
	writer.ObjectAttrs.ContentType = opts.GCSOptions.ObjectContentType

	return &Output{opts: opts, client: gcsClient, cancelFunc: cancel, writer: writer}, nil
}

// DialContext connects to the configured endpoint.
func (o *Output) DialContext(ctx context.Context) error {
	if err := o.createBucket(ctx); err != nil {
		return err
	}
	return nil
}

// Close closes the connection to the configured endpoint.
func (o *Output) Close() error {
	if err := o.writer.Close(); err != nil {
		return err
	}
	if err := o.client.Close(); err != nil {
		return err
	}
	o.cancelFunc()
	return nil
}

// Write writes data to the configured endpoint.
func (o *Output) Write(b []byte) (int, error) {
	if _, err := o.writer.Write(b); err != nil {
		return 0, fmt.Errorf("failed to copy data: %w", err)
	}

	return len(b), nil
}

func (o *Output) createBucket(ctx context.Context) error {
	bkt := o.client.Bucket(o.opts.GCSOptions.Bucket)
	_, err := bkt.Attrs(ctx)
	if errors.Is(err, storage.ErrBucketNotExist) {
		err = bkt.Create(ctx, o.opts.GCSOptions.ProjectID, nil)
		if err != nil {
			return fmt.Errorf("failed to create Bucket: %w", err)
		}
		return nil
	}
	return nil
}

// NewClient returns a new Google Cloud Storage client.
func NewClient(addr string) (gcsClient *storage.Client, ctx context.Context, cancel context.CancelFunc, err error) {
	ctx, cancel = context.WithCancel(context.Background())
	var h *url.URL
	if addr != "" {
		h, err = url.Parse(addr)
		if err != nil {
			return nil, nil, nil, err
		}
		h.Path = "storage/v1/"
		gcsClient, err = storage.NewClient(ctx, option.WithEndpoint(h.String()), option.WithoutAuthentication())
	} else {
		gcsClient, err = storage.NewClient(ctx)
	}
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("failed to create gcs client: %w", err)
	}

	return gcsClient, ctx, cancel, nil
}
