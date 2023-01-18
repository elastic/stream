// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package gcppubsub

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"google.golang.org/api/iterator"

	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("gcppubsub", New)
}

type Output struct {
	opts       *output.Options
	client     *pubsub.Client
	cancelFunc func()
}

func New(opts *output.Options) (output.Output, error) {
	if opts.Addr == "" {
		return nil, errors.New("emulator address is required")
	}

	os.Setenv("PUBSUB_EMULATOR_HOST", opts.Addr)

	ctx, cancel := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, opts.GCPPubsubOptions.Project)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &Output{opts: opts, client: client, cancelFunc: cancel}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	// Disable HTTP keep-alives to ensure no extra goroutines hang around.
	httpClient := http.Client{Transport: &http.Transport{DisableKeepAlives: true}}

	// Sanity check the emulator.
	resp, err := httpClient.Get("http://" + o.opts.Addr)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	if o.opts.GCPPubsubOptions.Clear {
		if err := o.clear(); err != nil {
			return err
		}
	}

	if err := o.createTopic(); err != nil {
		return err
	}

	if err := o.createSubscription(); err != nil {
		return err
	}

	return nil
}

func (o *Output) Close() error {
	o.client.Topic(o.opts.GCPPubsubOptions.Topic).Stop()
	o.cancelFunc()
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := o.client.Topic(o.opts.GCPPubsubOptions.Topic)
	result := topic.Publish(ctx, &pubsub.Message{Data: b})

	// Wait for message to publish and get assigned ID.
	if _, err := result.Get(ctx); err != nil {
		return 0, err
	}

	return len(b), nil
}

func (o *Output) clear() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Clear all topics.
	topics := o.client.Topics(ctx)
	for {
		topic, err := topics.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		if err = topic.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete topic %v: %w", topic.ID(), err)
		}
	}

	// Clear all subscriptions.
	subs := o.client.Subscriptions(ctx)
	for {
		sub, err := subs.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		if err = sub.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete subscription %v: %w", sub.ID(), err)
		}
	}

	return nil
}

func (o *Output) createTopic() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	topic := o.client.Topic(o.opts.GCPPubsubOptions.Topic)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if topic exists: %w", err)
	}

	if !exists {
		if _, err := o.client.CreateTopic(ctx, o.opts.GCPPubsubOptions.Topic); err != nil {
			return fmt.Errorf("failed to create the topic: %w", err)
		}
	}

	return nil
}

func (o *Output) createSubscription() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sub := o.client.Subscription(o.opts.GCPPubsubOptions.Subscription)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if sub exists: %w", err)
	}

	if !exists {
		_, err := o.client.CreateSubscription(
			ctx,
			o.opts.GCPPubsubOptions.Subscription,
			pubsub.SubscriptionConfig{
				Topic: o.client.Topic(o.opts.GCPPubsubOptions.Topic),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create subscription: %w", err)
		}
	}

	return nil
}
