// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.
package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/Shopify/sarama"
	"github.com/elastic/stream/pkg/output"
)

func init() {
	output.Register("kafka", New)
}

type Output struct {
	opts       *output.Options
	client     sarama.SyncProducer
	config     *sarama.Config
	cancelFunc func()
}

func New(opts *output.Options) (output.Output, error) {
	if opts.Addr == "" {
		return nil, errors.New("kafka address is required")
	}

	os.Setenv("KAFKA_HOST", opts.Addr)
	os.Setenv("KAFKA_TOPIC", opts.KafkaOptions.Topic)

	config := sarama.NewConfig()
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer([]string{opts.Addr}, config)
	_, cancel := context.WithCancel(context.Background())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return &Output{opts: opts, cancelFunc: cancel, client: producer, config: config}, nil
}

func (o *Output) DialContext(ctx context.Context) error {
	if err := o.createTopic(); err != nil {
		return err
	}

	return nil
}

func (o *Output) Close() error {
	o.cancelFunc()
	return nil
}

func (o *Output) Write(b []byte) (int, error) {
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	value := sarama.ByteEncoder(b)
	msg := &sarama.ProducerMessage{
		Topic: o.opts.KafkaOptions.Topic,
		Value: value}
	_, _, err := o.client.SendMessage(msg)

	if err != nil {
		return 0, fmt.Errorf("failed to create data in kafka topic: %v", err)
	}

	return value.Length(), nil
}

func (o *Output) createTopic() error {
	admin, err := sarama.NewClusterAdmin([]string{o.opts.Addr}, o.config)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}
	err = admin.CreateTopic(o.opts.KafkaOptions.Topic, &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}, false)

	if err != nil {
		return fmt.Errorf("failed to create topic: %v", err)
	}
	return nil
}
