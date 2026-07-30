// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package azureblobstorage provides an output for streaming data to Azure Blob
// Storage containers. This output implementation handles the creation of
// containers (if they do not exist) and uploads data as blobs using the Azure
// SDK for Go. Configuration options allow users to specify the container, blob
// name, and emulator port for testing.
package azureblobstorage

import (
	"context"
	"errors"
	"fmt"

	"github.com/elastic/stream/internal/output"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	blobalias "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
)

func init() {
	output.Register("azureblobstorage", New)
}

// Output is an Azure Blob Storage output.
type Output struct {
	opts   *output.Options
	client *azblob.Client
}

// New returns a new Azure Blob Storage output.
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

// DialContext connects to the configured endpoint.
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

// Write writes data to the Azure Blob Storage output.
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

// createContainer creates the container if it does not exist.
func (o *Output) createContainer(ctx context.Context) error {
	_, err := o.client.CreateContainer(ctx, o.opts.AzureBlobStorageOptions.Container, nil)
	if err != nil {
		return err
	}
	return nil
}
