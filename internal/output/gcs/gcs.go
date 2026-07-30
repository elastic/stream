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
	opts   *output.Options
	client *storage.Client
	writer *storage.Writer
}

// New returns a new GCS output.
func New(opts *output.Options) (output.Output, error) {
	return &Output{opts: opts}, nil
}

// DialContext connects to the configured endpoint.
func (o *Output) DialContext(ctx context.Context) error {
	gcsClient, err := NewClient(ctx, o.opts.Addr)
	if err != nil {
		return err
	}
	o.client = gcsClient

	if err := o.createBucket(ctx); err != nil {
		_ = gcsClient.Close()
		o.client = nil
		return err
	}

	obj := gcsClient.Bucket(o.opts.GCSOptions.Bucket).Object(o.opts.GCSOptions.Object)
	writer := obj.NewWriter(context.WithoutCancel(ctx))
	// System tests are failing because a default content type is not set automatically, so we set it here instead.
	writer.ObjectAttrs.ContentType = o.opts.GCSOptions.ObjectContentType
	o.writer = writer

	return nil
}

// Close closes the connection to the configured endpoint.
func (o *Output) Close() error {
	var closeErr error
	if o.writer != nil {
		closeErr = errors.Join(closeErr, o.writer.Close())
		o.writer = nil
	}
	if o.client != nil {
		closeErr = errors.Join(closeErr, o.client.Close())
		o.client = nil
	}
	return closeErr
}

// Write writes data to the configured endpoint.
func (o *Output) Write(b []byte) (int, error) {
	if o.writer == nil {
		return 0, errors.New("not connected")
	}

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

// NewClient returns a new Google Cloud Storage client using ctx.
func NewClient(ctx context.Context, addr string) (*storage.Client, error) {
	if addr != "" {
		endpoint, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		endpoint.Path = "storage/v1/"
		gcsClient, err := storage.NewClient(ctx, option.WithEndpoint(endpoint.String()), option.WithoutAuthentication())
		if err != nil {
			return nil, fmt.Errorf("failed to create gcs client: %w", err)
		}
		return gcsClient, nil
	}

	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcs client: %w", err)
	}
	return gcsClient, nil
}
