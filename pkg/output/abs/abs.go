// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package abs

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("abs", New)
}

type Output struct {
	opts       *output.Options
	client     *azblob.Client
	ctx        context.Context
	cancelFunc func()
}

func New(opts *output.Options) (output.Output, error) {
	if opts.Addr == "" {
		return nil, errors.New("azure blob storage address is required")
	}
	connectionString := fmt.Sprintf("DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://%s:%s/devstoreaccount1;", opts.Addr, opts.ABSOptions.Port)
	ctx, cancel := context.WithCancel(context.Background())
	serviceClient, _ := azblob.NewClientFromConnectionString(connectionString, nil)

	return &Output{opts: opts, client: serviceClient, ctx: ctx, cancelFunc: cancel}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	err := o.createContainer()
	if err != nil {
		return err
	}
	return nil
}

func (o *Output) Close() error {
	o.cancelFunc()
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	_, err := o.client.UploadBuffer(o.ctx, o.opts.ABSOptions.Container, o.opts.ABSOptions.Blob, b, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to upload file to blob: %w", err)
	}
	return len(b), nil
}

func (o *Output) createContainer() error {
	_, err := o.client.CreateContainer(o.ctx, o.opts.ABSOptions.Container, nil)
	if err != nil {
		return err
	}
	return nil
}
