// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package gcs

import (
	"context"
	"errors"
	"fmt"
	"os"

	"cloud.google.com/go/storage"

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
	if opts.Addr == "" {
		return nil, errors.New("google cloud address is required")
	}
	// https://cloud.google.com/go/docs/reference/cloud.google.com/go/storage/latest#hdr-Creating_a_Client
	// This is required to override the client to use localhost instead, has to be set before creating the client
	os.Setenv("STORAGE_EMULATOR_HOST", opts.Addr)
	defer os.Unsetenv("STORAGE_EMULATOR_HOST")

	ctx, cancel := context.WithCancel(context.Background())
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create gcs client: %w", err)
	}
	obj := gcsClient.Bucket(opts.GcsOptions.Bucket).Object(opts.GcsOptions.Object)
	writer := obj.NewWriter(ctx)

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
