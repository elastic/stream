// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package azureeventhub

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"

	"github.com/elastic/stream/internal/pkg/output"
)

func init() {
	output.Register("azureeventhub", New)
}

type Output struct {
	opts           *output.Options
	producerClient *azeventhubs.ProducerClient
	cancelFunc     context.CancelFunc
	cancelCtx      context.Context
}

func New(opts *output.Options) (output.Output, error) {
	var producerClient *azeventhubs.ProducerClient
	var err error

	if opts.AzureEventHubOptions.ConnectionString != "" {
		producerClient, err = azeventhubs.NewProducerClientFromConnectionString(opts.AzureEventHubOptions.ConnectionString, opts.AzureEventHubOptions.EventHubName, nil)
		if err != nil {
			return nil, fmt.Errorf("error while creating new eventhub producer client from connection string: %w", err)
		}
	} else {
		fmt.Print("no connection string was provided, falling back to default credentials or environment variable")

		// Credentials set as env variables - https://github.com/Azure/azure-sdk-for-go/tree/main/sdk/azidentity#environment-variables
		defaultAzureCred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("missing azure credentials in the environment variables: %w", err)
		}

		producerClient, err = azeventhubs.NewProducerClient(opts.AzureEventHubOptions.FullyQualifiedNamespace, opts.AzureEventHubOptions.EventHubName, defaultAzureCred, nil)
		if err != nil {
			return nil, fmt.Errorf("error while creating new eventhub producer client: %w", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Output{opts: opts, producerClient: producerClient, cancelFunc: cancel, cancelCtx: ctx}, nil
}

func (*Output) DialContext(_ context.Context) error {
	return nil
}

func (o *Output) Close() error {
	o.producerClient.Close(o.cancelCtx)
	o.cancelFunc()
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	batch, err := o.producerClient.NewEventDataBatch(o.cancelCtx, nil)
	if err != nil {
		return 0, fmt.Errorf("error while creating new event data batch: %w", err)
	}
	eventData := azeventhubs.EventData{Body: b}

	if err := batch.AddEventData(&eventData, nil); err != nil {
		return 0, fmt.Errorf("error while adding data to event data batch: %w", err)
	}

	if err := o.producerClient.SendEventDataBatch(context.TODO(), batch, nil); err != nil {
		return 0, fmt.Errorf("error while sending event data batch: %w", err)
	}

	return len(b), nil
}
