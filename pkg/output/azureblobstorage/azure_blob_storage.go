// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package azureblobstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/stream/pkg/output"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	blobalias "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

func init() {
	output.Register("azureblobstorage", New)
}

type Output struct {
	opts   *output.Options
	client *azblob.Client
}

func New(opts *output.Options) (output.Output, error) {
	if opts.Addr == "" {
		return nil, errors.New("azure blob storage address is required")
	}
	// A connection string is used for multiple reasons, its easier to bypass the URL endpoint, and the hardcoded credentials can easily be passed.
	// These credentials are the defaults for the Azurite Emulator, which is why they can simply be hardcoded.
	connectionString := fmt.Sprintf("DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://%s:%s/devstoreaccount1;", opts.Addr, opts.AzureBlobStorageOptions.Port)
	serviceClient, _ := azblob.NewClientFromConnectionString(connectionString, nil)

	return &Output{opts: opts, client: serviceClient}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	if err := o.createContainer(ctx); err != nil {
		return err
	}
	return nil
}

// Close is not needed as there is no client to close
func (*Output) Close() error {
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	cType := "application/json"
	options := azblob.UploadBufferOptions{
		HTTPHeaders: &blobalias.HTTPHeaders{
			BlobContentType: &cType,
		},
	}
	_, err := o.client.UploadBuffer(context.Background(), o.opts.AzureBlobStorageOptions.Container, o.opts.AzureBlobStorageOptions.Blob, b, &options)
	if err != nil {
		return 0, fmt.Errorf("failed to upload file to blob: %w", err)
	}
	return len(b), nil
}

func (o *Output) createContainer(ctx context.Context) error {
	_, err := o.client.CreateContainer(ctx, o.opts.AzureBlobStorageOptions.Container, nil)
	if err != nil {
		return err
	}
	return nil
}
