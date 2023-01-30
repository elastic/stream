// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("gcs", New)
}

type Output struct {
	opts       *output.Options
	client     *storage.Client
	writer     *storage.Writer
	cancelFunc func()
}

func New(opts *output.Options) (output.Output, error) {
	gcsClient, ctx, cancel, err := NewClient(opts.Addr)
	if err != nil {
		return nil, err
	}
	obj := gcsClient.Bucket(opts.GcsOptions.Bucket).Object(opts.GcsOptions.Object)
	writer := obj.NewWriter(ctx)
	// System tests are failing because a default content type is not set automatically, so we set it here instead.
	writer.ObjectAttrs.ContentType = opts.GcsOptions.ObjectContentType

	return &Output{opts: opts, client: gcsClient, cancelFunc: cancel, writer: writer}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	if err := o.createBucket(ctx); err != nil {
		return err
	}
	return nil
}

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

func (o *Output) Write(b []byte) (int, error) {
	if _, err := o.writer.Write(b); err != nil {
		return 0, fmt.Errorf("failed to copy data: %w", err)
	}

	return len(b), nil
}

func (o *Output) createBucket(ctx context.Context) error {
	bkt := o.client.Bucket(o.opts.GcsOptions.Bucket)
	_, err := bkt.Attrs(ctx)
	if errors.Is(err, storage.ErrBucketNotExist) {
		err = bkt.Create(ctx, o.opts.GcsOptions.ProjectID, nil)
		if err != nil {
			return fmt.Errorf("failed to create Bucket: %w", err)
		}
		return nil
	}
	return nil
}

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
