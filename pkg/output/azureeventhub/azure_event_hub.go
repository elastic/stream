// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package azureeventhub

import (
	"context"
	"fmt"

	"github.com/elastic/stream/pkg/output"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
)

func init() {
	output.Register("azureeventhub", New)
}

type Output struct {
	opts           *output.Options
	producerClient *azeventhubs.ProducerClient
}

func New(opts *output.Options) (output.Output, error) {
	// Credentials set as env variables - https://github.com/Azure/azure-sdk-for-go/blob/6b6f76ebe0d2334c83e8b6f89af4fe9d0b1ce631/sdk/azidentity/README.md?plain=1#L156-L187
	defaultAzureCred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("missing azure credentials in the environment variables : %w", err)
	}

	producerClient, err := azeventhubs.NewProducerClient(opts.AzureEventHubOptions.FullyQualifiedNamespace, opts.AzureEventHubOptions.EventHubName, defaultAzureCred, nil)
	if err != nil {
		return nil, fmt.Errorf("error while creating new eventhub producer client : %w", err)
	}

	return &Output{opts: opts, producerClient: producerClient}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	return nil
}

func (o *Output) Close() error {
	o.producerClient.Close(context.TODO())
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	batch, err := o.producerClient.NewEventDataBatch(context.TODO(), nil)
	if err != nil {
		return 0, fmt.Errorf("error while creating new event data batch : %w", err)
	}
	eventData := azeventhubs.EventData{Body: b}

	if err := batch.AddEventData(&eventData, nil); err != nil {
		return 0, fmt.Errorf("error while adding data to event data batch : %w", err)
	}

	if err := o.producerClient.SendEventDataBatch(context.TODO(), batch, nil); err != nil {
		return 0, fmt.Errorf("error while sending event data batch : %w", err)
	}

	return len(b), nil
}
