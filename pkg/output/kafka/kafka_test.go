// Licensed to Elasticsearch B.V. under one or more agreements.
// Elasticsearch B.V. licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package kafka

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/elastic/stream/pkg/output"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

const (
	emulatorHost  = "127.0.0.1"
	emulatorPort  = "9092"
	topic         = "testTopic"
	consumerGroup = "testGroup"
)

var (
	outputevent string
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "bashj79/kafka-kraft",
		Tag:        "latest",
		PortBindings: map[docker.Port][]docker.PortBinding{
			emulatorPort: {{HostIP: emulatorHost, HostPort: emulatorPort}},
		},
		ExposedPorts: []string{emulatorPort},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}
	if err := pool.Retry(func() error {
		config := sarama.NewConfig()
		config.Producer.Partitioner = sarama.NewRandomPartitioner
		config.Producer.RequiredAcks = sarama.WaitForAll
		config.Producer.Return.Successes = true
		_, err := sarama.NewClient([]string{fmt.Sprintf("%s:%s", emulatorHost, emulatorPort)}, config)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("Could not connect to the kafka instance: %s", err)
	}

	code := m.Run()

	_ = pool.Purge(resource)

	os.Exit(code)
}

func TestKafka(t *testing.T) {
	out, err := New(&output.Options{
		Addr: fmt.Sprintf("%s:%s", emulatorHost, emulatorPort),
		KafkaOptions: output.KafkaOptions{
			Topic: topic,
		},
	})
	require.NoError(t, err)

	event := "testmessage something"
	require.NoError(t, err)

	n, err := out.Write([]byte(event))
	require.NoError(t, err)
	assert.Equal(t, len(event), n)
	consumer, err := sarama.NewConsumer(strings.Split(fmt.Sprintf("%s:%s", emulatorHost, emulatorPort), ","), nil)

	require.NoError(t, err)

	partitionList, err := consumer.Partitions(topic)
	require.NoError(t, err)

	for partition := range partitionList {
		pc, err := consumer.ConsumePartition(topic, int32(partition), sarama.OffsetOldest)
		if err != nil {
			fmt.Printf("Failed to start consumer for partition %d: %s\n", partition, err)
			return
		}
		for msg := range pc.Messages() {
			outputevent = string(msg.Value)
			break
		}
	}
	assert.Equal(t, event, outputevent)
	t.Cleanup(func() { _ = consumer.Close() })
}
