// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

// Package kafka provides an output implementation for streaming data to Apache
// Kafka topics. It enables sending events or log lines from the stream utility
// to a specified Kafka cluster, handling connection setup, message production,
// and topic management. This output supports configurable options such as target
// topic and broker address.
package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/IBM/sarama"

	"github.com/elastic/stream/internal/output"
)

func init() {
	output.Register("kafka", New)
}

// Output is a kafka output.
type Output struct {
	opts   *output.Options
	client sarama.SyncProducer
	config *sarama.Config
}

// New returns a new kafka output.
func New(opts *output.Options) (output.Output, error) {
	if opts.Addr == "" {
		return nil, errors.New("kafka address is required")
	}

	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	saramaClient, err := sarama.NewClient([]string{opts.Addr}, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create sarama client: %w", err)
	}

	producer, err := sarama.NewSyncProducerFromClient(saramaClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer client: %w", err)
	}

	return &Output{opts: opts, client: producer, config: config}, nil
}

// DialContext connects to the configured endpoint.
func (o *Output) DialContext(_ context.Context) error {
	if err := o.createTopic(); err != nil {
		return err
	}

	return nil
}

// Close closes the connection to the configured endpoint.
func (o *Output) Close() error {
	o.client.Close()
	return nil
}

// Write writes data to the kafka topic.
func (o *Output) Write(b []byte) (int, error) {
	msg := &sarama.ProducerMessage{
		Topic: o.opts.KafkaOptions.Topic,
		Value: sarama.ByteEncoder(b),
	}
	_, _, err := o.client.SendMessage(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to create data in kafka topic: %w", err)
	}

	return len(b), nil
}

func (o *Output) createTopic() error {
	admin, err := sarama.NewClusterAdmin([]string{o.opts.Addr}, o.config)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin client: %w", err)
	}

	err = admin.CreateTopic(o.opts.KafkaOptions.Topic, &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, false)

	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}
	return nil
}
